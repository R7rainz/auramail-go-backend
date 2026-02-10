package calendar

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/r7rainz/auramail/internal/auth"
	"github.com/r7rainz/auramail/internal/auth/google"
	"github.com/r7rainz/auramail/internal/response"
	"github.com/r7rainz/auramail/internal/user"
	gcalendar "google.golang.org/api/calendar/v3"
)

type Handler struct {
	userRepo *user.PostgresRepository
}

func NewHandler(repo *user.PostgresRepository) *Handler {
	return &Handler{
		userRepo: repo,
	}
}

// AddEventRequest represents the request body for adding a calendar event
type AddEventRequest struct {
	Title       string `json:"title"`       // Event title (e.g., "Microsoft Interview")
	Description string `json:"description"` // Full description/details
	StartTime   string `json:"startTime"`   // ISO 8601 format: "2026-02-15T10:00:00"
	EndTime     string `json:"endTime"`     // ISO 8601 format: "2026-02-15T11:00:00"
	Location    string `json:"location"`    // Optional location
	EmailID     string `json:"emailId"`     // Reference to the email summary
	Company     string `json:"company"`     // Company name for context
	Role        string `json:"role"`        // Role for context
	EventType   string `json:"eventType"`   // "deadline", "interview", "exam", "event"
}

// AddEventResponse represents the response after adding a calendar event
type AddEventResponse struct {
	Success   bool   `json:"success"`
	EventID   string `json:"eventId"`
	EventLink string `json:"eventLink"`
	Message   string `json:"message"`
}

// AddEvent adds a placement-related event to the user's Google Calendar
func (h *Handler) AddEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("AddEvent called")

	userID, ok := ctx.Value(auth.UserIDContextKey).(int)
	if !ok {
		slog.Error("No UserID in context")
		response.Unauthorized(w, "No UserID found in context")
		return
	}

	// Parse request body
	var req AddEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request body", "err", err)
		response.BadRequest(w, "Invalid request body", nil)
		return
	}

	slog.Info("Calendar event request", "title", req.Title, "startTime", req.StartTime, "userID", userID)

	// Validate required fields
	if req.Title == "" || req.StartTime == "" {
		slog.Error("Missing required fields", "title", req.Title, "startTime", req.StartTime)
		response.BadRequest(w, "Title and startTime are required", nil)
		return
	}

	// Get user to retrieve refresh token
	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		slog.Error("User not found", "userID", userID, "err", err)
		response.NotFound(w, "User not found")
		return
	}

	slog.Info("Found user, creating calendar service", "email", u.Email, "hasRefreshToken", u.RefreshToken != "")

	// Create calendar service
	calSvc, err := google.CreateCalendarService(ctx, u.RefreshToken)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create calendar service",
			"err", err,
			"userID", userID,
			"hasRefreshToken", u.RefreshToken != "",
		)
		response.InternalError(w, "Failed to connect to Google Calendar. You may need to re-login to grant calendar permissions.")
		return
	}

	slog.Info("Calendar service created successfully")

	// Parse start time
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		slog.Info("Trying alternative date format", "original", req.StartTime)
		// Try parsing without timezone
		startTime, err = time.Parse("2006-01-02T15:04:05", req.StartTime)
		if err != nil {
			// Try date only format
			startTime, err = time.Parse("2006-01-02", req.StartTime)
			if err != nil {
				slog.Error("Failed to parse startTime", "startTime", req.StartTime, "err", err)
				response.BadRequest(w, "Invalid startTime format. Use ISO 8601 format (e.g., 2026-02-15T10:00:00 or 2026-02-15)", nil)
				return
			}
			// If date only, set time to 10:00 AM
			startTime = startTime.Add(10 * time.Hour)
		}
	}

	slog.Info("Parsed start time", "startTime", startTime)

	// Set end time (default to 1 hour after start if not provided)
	var endTime time.Time
	if req.EndTime != "" {
		endTime, err = time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			endTime, err = time.Parse("2006-01-02T15:04:05", req.EndTime)
			if err != nil {
				response.BadRequest(w, "Invalid endTime format. Use ISO 8601 format", nil)
				return
			}
		}
	} else {
		endTime = startTime.Add(1 * time.Hour)
	}

	// Build description with context
	description := req.Description
	if req.Company != "" || req.Role != "" {
		description = "Company: " + req.Company + "\n"
		if req.Role != "" {
			description += "Role: " + req.Role + "\n"
		}
		description += "\n" + req.Description
	}
	description += "\n\n---\nAdded via AuraMail"

	// Create the calendar event
	event := &gcalendar.Event{
		Summary:     req.Title,
		Description: description,
		Location:    req.Location,
		Start: &gcalendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "Asia/Kolkata", // Indian timezone for VIT students
		},
		End: &gcalendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "Asia/Kolkata",
		},
		Reminders: &gcalendar.EventReminders{
			UseDefault:      false,
			ForceSendFields: []string{"UseDefault"},
			Overrides: []*gcalendar.EventReminder{
				{Method: "popup", Minutes: 30},   // 30 min before
				{Method: "popup", Minutes: 60},   // 1 hour before
				{Method: "email", Minutes: 1440}, // 1 day before
			},
		},
		// Add color based on event type
		ColorId: getColorForEventType(req.EventType),
	}

	// Insert the event
	slog.Info("Inserting calendar event",
		"summary", event.Summary,
		"startDateTime", event.Start.DateTime,
		"endDateTime", event.End.DateTime,
	)
	createdEvent, err := calSvc.Events.Insert("primary", event).Do()
	if err != nil {
		slog.ErrorContext(ctx, "failed to create calendar event",
			"err", err,
			"errorType", fmt.Sprintf("%T", err),
			"userID", userID,
			"summary", event.Summary,
		)
		response.InternalError(w, fmt.Sprintf("Failed to create calendar event: %v", err))
		return
	}

	slog.Info("calendar event created",
		"eventId", createdEvent.Id,
		"title", req.Title,
		"userId", userID,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AddEventResponse{
		Success:   true,
		EventID:   createdEvent.Id,
		EventLink: createdEvent.HtmlLink,
		Message:   "Event added to your Google Calendar with reminders",
	})
}

// getColorForEventType returns a Google Calendar color ID based on event type
// Color IDs: 1=Lavender, 2=Sage, 3=Grape, 4=Flamingo, 5=Banana, 6=Tangerine, 7=Peacock, 8=Graphite, 9=Blueberry, 10=Basil, 11=Tomato
func getColorForEventType(eventType string) string {
	switch eventType {
	case "deadline":
		return "11" // Tomato (red) - urgent
	case "interview":
		return "7" // Peacock (teal) - important
	case "exam":
		return "6" // Tangerine (orange) - attention
	case "event":
		return "9" // Blueberry (blue) - informational
	default:
		return "1" // Lavender (default)
	}
}

// DeleteEvent removes a calendar event (in case user wants to unmark)
func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(auth.UserIDContextKey).(int)
	if !ok {
		response.Unauthorized(w, "No UserID found in context")
		return
	}

	// Get event ID from query param
	eventID := r.URL.Query().Get("eventId")
	if eventID == "" {
		response.BadRequest(w, "eventId is required", nil)
		return
	}

	// Get user
	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		response.NotFound(w, "User not found")
		return
	}

	// Create calendar service
	calSvc, err := google.CreateCalendarService(ctx, u.RefreshToken)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create calendar service", "err", err)
		response.InternalError(w, "Failed to connect to Google Calendar")
		return
	}

	// Delete the event
	err = calSvc.Events.Delete("primary", eventID).Do()
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete calendar event", "err", err, "eventId", eventID)
		response.InternalError(w, "Failed to delete calendar event")
		return
	}

	slog.Info("calendar event deleted", "eventId", eventID, "userId", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Event removed from your Google Calendar",
	})
}

// CalendarEvent represents a calendar event for the frontend
type CalendarEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Location    string `json:"location"`
	Link        string `json:"link"`
	ColorID     string `json:"colorId"`
	IsAuraMail  bool   `json:"isAuraMail"` // true if added via AuraMail
}

// GetEvents fetches upcoming calendar events for the user
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(auth.UserIDContextKey).(int)
	if !ok {
		response.Unauthorized(w, "No UserID found in context")
		return
	}

	// Get user
	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		response.NotFound(w, "User not found")
		return
	}

	// Create calendar service
	calSvc, err := google.CreateCalendarService(ctx, u.RefreshToken)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create calendar service", "err", err)
		response.InternalError(w, "Failed to connect to Google Calendar")
		return
	}

	// Get time range - default to next 30 days
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}

	now := time.Now()
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 0, days).Format(time.RFC3339)

	// Fetch events from Google Calendar
	events, err := calSvc.Events.List("primary").
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(50).
		Do()

	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch calendar events", "err", err)
		response.InternalError(w, "Failed to fetch calendar events")
		return
	}

	// Transform to our format
	calendarEvents := make([]CalendarEvent, 0, len(events.Items))
	for _, item := range events.Items {
		// Determine start/end times
		startTime := ""
		endTime := ""
		if item.Start != nil {
			if item.Start.DateTime != "" {
				startTime = item.Start.DateTime
			} else {
				startTime = item.Start.Date
			}
		}
		if item.End != nil {
			if item.End.DateTime != "" {
				endTime = item.End.DateTime
			} else {
				endTime = item.End.Date
			}
		}

		// Check if this event was added via AuraMail
		isAuraMail := false
		if item.Description != "" && len(item.Description) > 20 {
			isAuraMail = strings.Contains(item.Description, "Added via AuraMail")
		}

		calendarEvents = append(calendarEvents, CalendarEvent{
			ID:          item.Id,
			Title:       item.Summary,
			Description: item.Description,
			StartTime:   startTime,
			EndTime:     endTime,
			Location:    item.Location,
			Link:        item.HtmlLink,
			ColorID:     item.ColorId,
			IsAuraMail:  isAuraMail,
		})
	}

	slog.Info("fetched calendar events", "count", len(calendarEvents), "userId", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"events":  calendarEvents,
		"total":   len(calendarEvents),
	})
}
