package gmail

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/r7rainz/auramail/internal/ai"
	"github.com/r7rainz/auramail/internal/auth"
	"github.com/r7rainz/auramail/internal/auth/google"
	"github.com/r7rainz/auramail/internal/config"
	"github.com/r7rainz/auramail/internal/response"
	"github.com/r7rainz/auramail/internal/user"
)

type GmailHandler struct {
	userRepo *user.PostgresRepository
	cfg      *config.Config
}

func NewHandler(cfg *config.Config, repo *user.PostgresRepository) *GmailHandler {
	return &GmailHandler{
		userRepo: repo,
		cfg:      cfg,
	}
}

// GetEmails returns stored email summaries for the authenticated user
func (h *GmailHandler) GetEmails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(auth.UserIDContextKey).(int)
	if !ok {
		response.Unauthorized(w, "No UserID found in context")
		return
	}

	// Get optional query params
	searchQuery := r.URL.Query().Get("q")
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Fetch stored summaries from database
	summaries, err := h.userRepo.GetSummariesByQuery(ctx, userID, searchQuery)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch email summaries", "err", err)
		response.InternalError(w, "Failed to fetch emails")
		return
	}

	// Transform to frontend expected format
	type emailResponse struct {
		ID                string   `json:"id"`
		GmailMessageID    string   `json:"gmailMessageId"`
		Subject           string   `json:"subject"`
		Sender            string   `json:"sender"`
		Snippet           string   `json:"snippet"`
		ReceivedAt        string   `json:"receivedAt"`
		Company           *string  `json:"company"`
		Role              *string  `json:"role"`
		Deadline          *string  `json:"deadline"`
		ApplyLink         *string  `json:"applyLink"`
		OtherLinks        []string `json:"otherLinks"`
		Eligibility       any      `json:"eligibility"`
		Timings           any      `json:"timings"`
		Salary            any      `json:"salary"`
		Location          any      `json:"location"`
		EventDetails      any      `json:"eventDetails"`
		Requirements      any      `json:"requirements"`
		Description       *string  `json:"description"`
		AttachmentSummary *string  `json:"attachmentSummary"`
		Category          string   `json:"category"`
		Tags              []string `json:"tags"`
		Priority          string   `json:"priority"`
		Summary           string   `json:"summary"`
	}

	emails := make([]emailResponse, 0, len(summaries))
	for i, s := range summaries {
		emails = append(emails, emailResponse{
			ID:                strconv.Itoa(i + 1),
			GmailMessageID:    "", // Would need to store this in DB
			Subject:           s.Summary,
			Sender:            "",
			Snippet:           s.Summary,
			ReceivedAt:        time.Now().Format(time.RFC3339), // Would need actual time from DB
			Company:           s.Company,
			Role:              s.Role,
			Deadline:          s.Deadline,
			ApplyLink:         s.ApplyLink,
			OtherLinks:        s.OtherLinks,
			Eligibility:       s.Eligibility,
			Timings:           s.Timings,
			Salary:            s.Salary,
			Location:          s.Location,
			EventDetails:      s.EventDetails,
			Requirements:      s.Requirements,
			Description:       s.Description,
			AttachmentSummary: s.AttachmentSummary,
			Category:          s.Category,
			Tags:              s.Tags,
			Priority:          s.Priority,
			Summary:           s.Summary,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"emails":     emails,
		"total":      len(emails),
		"page":       page,
		"totalPages": 1,
	})
}

func (h *GmailHandler) SyncPlacementEmails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(auth.UserIDContextKey).(int)
	if !ok {
		response.Unauthorized(w, "No UserID found in context")
		return
	}

	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		response.NotFound(w, "User not found")
		return
	}

	srv, err := google.CreateGmailService(ctx, u.RefreshToken)
	if err != nil {
		slog.ErrorContext(ctx, "gmail service init failed", "err", err)
		response.InternalError(w, "Failed to connect to Gmail")
		return
	}

	// Use configurable query with fallback
	query := r.URL.Query().Get("query")
	if query == "" {
		query = h.cfg.DefaultEmailQuery
	}

	slog.Info("Starting email sync with AI processing", "query", query, "userID", userID)

	// Use FetchAndSummarize to get AI-processed emails
	emailStream := FetchAndSummarize(ctx, srv, h.userRepo, query, u.ID)

	// Collect all processed emails
	var processedEmails []*ai.AIResult
	for summary := range emailStream {
		if summary != nil {
			processedEmails = append(processedEmails, summary)
			slog.Info("Processed email", "company", summary.Company, "category", summary.Category)
		}
	}

	slog.Info("Email sync completed", "processed", len(processedEmails))

	response.Success(w, map[string]interface{}{
		"success":   true,
		"processed": len(processedEmails),
		"emails":    processedEmails,
	})
}

func (h *GmailHandler) StreamPlacementEmails(w http.ResponseWriter, r *http.Request) {
	// Setting headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // for development

	ctx := r.Context()

	rawID := ctx.Value(auth.UserIDContextKey)
	if rawID == nil {
		sendSSEError(w, "unauthorized", "No UserID in context")
		return
	}

	userID, ok := rawID.(int)
	if !ok {
		sendSSEError(w, "unauthorized", "Invalid UserID type")
		return
	}

	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		sendSSEError(w, "not_found", "User not found")
		return
	}

	srv, err := google.CreateGmailService(ctx, u.RefreshToken)
	if err != nil {
		sendSSEError(w, "auth_error", "Failed to initialize Gmail service")
		return
	}

	// Use query param or configurable default
	query := r.URL.Query().Get("query")
	if query == "" {
		query = h.cfg.DefaultEmailQuery
	}

	// Send cached history first for instant UI update
	history, _ := h.userRepo.GetSummariesByQuery(ctx, u.ID, "")
	for _, hist := range history {
		jsonData, _ := json.Marshal(hist)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
	}
	w.(http.Flusher).Flush()

	// Start live stream for new emails
	emailStream := FetchAndSummarize(ctx, srv, h.userRepo, query, u.ID)

	foundAny := false

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// User closed connection
			return
		case summary, ok := <-emailStream:
			if !ok {
				// Channel closed
				if !foundAny {
					sendSSEError(w, "no_emails", "No emails found matching the query")
				}
				// Send completion event
				fmt.Fprintf(w, "event: complete\ndata: {\"status\": \"done\"}\n\n")
				w.(http.Flusher).Flush()
				return
			}

			foundAny = true
			jsonData, err := json.Marshal(summary)
			if err != nil {
				log.Printf("Error marshaling AI result: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			w.(http.Flusher).Flush()
		case <-ticker.C:
			// Heartbeat to keep connection alive
			fmt.Fprintf(w, ": heartbeat\n\n")
			w.(http.Flusher).Flush()
		}
	}
}

// sendSSEError sends an error message via SSE
func sendSSEError(w http.ResponseWriter, code, message string) {
	errData := map[string]string{
		"error":   code,
		"message": message,
	}
	jsonData, _ := json.Marshal(errData)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", jsonData)
	w.(http.Flusher).Flush()
}
