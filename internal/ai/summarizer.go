package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

type AIResult struct {
	Summary           string   `json:"summary"`
	Category          string   `json:"category"`
	Company           *string  `json:"company"`
	Role              *string  `json:"role"`
	Deadline          *string  `json:"deadline"`
	ApplyLink         *string  `json:"applyLink"`
	OtherLinks        []string `json:"otherLinks"`
	Eligibility       *string  `json:"eligibility"`
	Timings           *string  `json:"timings"`
	Salary            *string  `json:"salary"`
	Location          *string  `json:"location"`
	EventDetails      *string  `json:"eventDetails"`
	Requirements      *string  `json:"requirements"`
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

const CACHE_TTL = 1 * time.Hour

func getClient() *openai.Client {
	once.Do(func() {
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey != "" {
			client = openai.NewClient(apiKey)
		}
	})
	return client
}

func AnalyzeEmail(ctx context.Context, subject, snippet, body string) (*AIResult, error) {
	cacheKey := fmt.Sprintf("%s:%s", subject, snippet)
	if len(cacheKey) > 100 {
		cacheKey = cacheKey[:100]
	}

	cacheMu.RLock()
	cached, exists := aiCache[cacheKey]
	cacheMu.RUnlock()

	if exists && time.Since(cached.timestamp) < CACHE_TTL {
		return cached.data, nil
	}

	truncatedBody := body
	if len(body) > 5000 {
		truncatedBody = body[:5000] + "..."
	}

	prompt := fmt.Sprintf(`Analyze this email and return ONLY a JSON object.
Subject: %s
Snippet: %s
Body: %s

(Follow the strict JSON format with fields: summary, category, company, role, deadline, applyLink, otherLinks, eligibility, timings, salary, location, eventDetails, requirements, description, attachmentSummary)`,
		subject, snippet, truncatedBody)

	c := getClient()
	if c == nil {
		return &AIResult{Summary: subject, Category: "misc"}, nil
	}

	resp, err := c.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are an expert student assistant. Return valid JSON only."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.05,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})

	if err != nil {
		return nil, err
	}

	var result AIResult
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	aiCache[cacheKey] = cacheItem{data: &result, timestamp: time.Now()}
	cacheMu.Unlock()

	return &result, nil
}
