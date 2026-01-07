package gmail

import (
	"context"
	"log"
	"sync"

	"google.golang.org/api/gmail/v1"

	"github.com/r7rainz/auramail/internal/ai"
	"github.com/r7rainz/auramail/internal/utils"
)

func FetchAndSummarize(ctx context.Context, srv *gmail.Service, query string, userID int) chan *ai.AIResult {
	out := make(chan *ai.AIResult)

	go func() {
		defer close(out)

		// 1. Safety check: Ensure list is not nil
		list, err := srv.Users.Messages.List("me").Q(query).MaxResults(10).Do()
		if err != nil || list == nil || len(list.Messages) == 0 {
			log.Printf("No messages found or error: %v", err)
			return
		}

		var wg sync.WaitGroup
		jobs := make(chan string, len(list.Messages))

		// 2. Start workers
		workerCount := 5 // 10 might hit OpenAI rate limits too fast, 5 is safer
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for id := range jobs {
					msg, err := srv.Users.Messages.Get("me", id).Format("full").Do()
					if err != nil {
						continue
					}
					
					subject := ""
					for _, h := range msg.Payload.Headers {
						if h.Name == "Subject" {
							subject = h.Value
						}
					}

					body := utils.ParseBody(msg.Payload)

					// 3. Summarize and Validate
					summary, err := ai.AnalyzeEmail(ctx, userID, subject, msg.Snippet, body)
					if err != nil || summary == nil {
						log.Printf("Skipping empty summary for %s: %v", id, err)
						continue
					}
					
					// Only send if we have a valid result
					select {
					case <-ctx.Done():
						return
					case out <- summary:
					}
				}
			}()
		}

		// 4. Feed the jobs
		for _, m := range list.Messages {
			jobs <- m.Id
		}
		close(jobs)
		
		// 5. Wait for completion
		wg.Wait()
	}()

	return out
}
