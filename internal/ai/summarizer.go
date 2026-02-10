package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

type AIResult struct {
	Summary           string   `json:"summary"`
	Category          string   `json:"category"`
	Tags              []string `json:"tags"`        // New: multiple tags for filtering
	Priority          string   `json:"priority"`    // New: high, medium, low
	Company           *string  `json:"company"`     // Pointer handles "null"
	Role              *string  `json:"role"`        // Pointer handles "null"
	Deadline          *string  `json:"deadline"`    // Pointer handles "null"
	ApplyLink         *string  `json:"applyLink"`   // Pointer handles "null"
	OtherLinks        []string `json:"otherLinks"`  // Slice handles []
	Eligibility       any      `json:"eligibility"` // 'any' is safest for bullet points
	Timings           any      `json:"timings"`     // 'any' is safest for bullet points
	Salary            any      `json:"salary"`      // 'any' is safest for bullet points
	Location          any      `json:"location"`    // 'any' is safest for bullet points
	EventDetails      any      `json:"eventDetails"`
	Requirements      any      `json:"requirements"`
	Description       *string  `json:"description"`
	AttachmentSummary *string  `json:"attachmentSummary"`
}

type cacheItem struct {
	data      *AIResult
	timestamp time.Time
}

var (
	aiCache = make(map[string]cacheItem)
	cacheMu sync.RWMutex // protects the map from concurrent access
	client  *openai.Client
	once    sync.Once
)

const CacheTTL = 1 * time.Hour

func getClient() *openai.Client {
	once.Do(func() {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey != "" {
			client = openai.NewClient(apiKey)
		}
	})
	return client
}

func AnalyzeEmail(ctx context.Context, userID int, subject, snippet, body string) (*AIResult, error) {
	cacheKey := fmt.Sprintf("user:%d:%s:%s", userID, subject, snippet)
	if len(cacheKey) > 100 {
		cacheKey = cacheKey[:100]
	}

	cacheMu.RLock()
	cached, exists := aiCache[cacheKey]
	cacheMu.RUnlock()
	if exists && time.Since(cached.timestamp) < CacheTTL {
		return cached.data, nil
	}

	truncatedBody := body
	if len(body) > 4000 { // Reduced slightly to leave room for the heavy prompt
		truncatedBody = body[:4000] + "..."
	}

	systemPrompt := `You are a highly specialized AI assistant for academic and recruitment analysis at VIT (Vellore Institute of Technology).
Return ONLY a valid JSON object.

CATEGORIZATION RULES (category field - pick the MOST SPECIFIC one):
- "internship" - Internship opportunities, summer internships, intern positions
- "job offer" - Full-time job offers, placement offers, FTE positions
- "ppt" - Pre-Placement Talks, company presentations, PPT schedules
- "workshop" - Workshops, bootcamps, training sessions, hackathons
- "exam" - Online assessments, tests, coding rounds, aptitude tests
- "interview" - Interview schedules, interview calls, HR rounds
- "result" - Results announcements, shortlists, selection lists
- "reminder" - Deadline reminders, follow-ups, last date notices
- "announcement" - General placement announcements, policy updates
- "registration" - Registration links, sign-up forms, application deadlines

TAGGING RULES (tags field - array of relevant tags):
Include ALL applicable tags from: ["urgent", "high-package", "dream-company", "mass-hiring", "off-campus", "on-campus", "remote", "hybrid", "wfh", "tier-1", "startup", "mnc", "govt", "psu", "core", "it", "non-tech", "fresher-friendly"]

JSON FIELD RULES:
- deadline: Use YYYY-MM-DD format or null.
- otherLinks: Must be an array of strings [].
- tags: Must be an array of strings [].
- eligibility, timings, salary, location, eventDetails, requirements: Must be a single string with \n• bullet points.
- company, role, applyLink, description, attachmentSummary: Use a string or null.
- If data is missing, use null (not empty string).
- priority: "high" if deadline within 3 days or dream company, "medium" if within a week, "low" otherwise.`

	userPrompt := fmt.Sprintf("Subject: %s\nSnippet: %s\nBody: %s", subject, snippet, truncatedBody)

	c := getClient()
	if c == nil {
		return &AIResult{Summary: subject, Category: "misc"}, nil
	}

	resp, err := c.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.1, // Low temperature for higher consistency
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("openai error: %w", err)
	}

	var result AIResult
	content := resp.Choices[0].Message.Content

	// Unmarshal directly into your pointer-ready struct
	err = json.Unmarshal([]byte(content), &result)
	if err != nil {
		log.Printf("JSON Unmarshal error: %v | Content: %s", err, content)
		return nil, err
	}

	cacheMu.Lock()
	aiCache[cacheKey] = cacheItem{data: &result, timestamp: time.Now()}
	cacheMu.Unlock()

	return &result, nil
}
