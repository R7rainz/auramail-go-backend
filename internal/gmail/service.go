package gmail

import (
	"context"
	"sync"

	"google.golang.org/api/gmail/v1"
    "github.com/r7rainz/auramail/internal/utils"
	"github.com/r7rainz/auramail/internal/ai"
)

func FetchAndSummarize(ctx context.Context, srv *gmail.Service, query string) chan *ai.AIResult {
	out := make(chan *ai.AIResult)
	list, _ := srv.Users.Messages.List("me").Q(query).MaxResults(10).Do()

	go func(){
		defer close(out)
		var wg sync.WaitGroup
		jobs := make(chan string, len(list.Messages))

		//Start 10 workers
		for range 10 {
			wg.Go(func() {
				for id:= range jobs {
					msg, err := srv.Users.Messages.Get("me", id).Format("full").Do()
					if err != nil {
						continue
					}
					subject := ""
					for _, h:= range msg.Payload.Headers {
						if h.Name == "Subject" {
							subject = h.Value
						}
					}

					body:= utils.ParseBody(msg.Payload)

					summary, _ := ai.AnalyzeEmail(ctx, subject, msg.Snippet, body)
					out <- summary
				}
			} )
		}
		for _, m := range list.Messages {
			jobs <- m.Id
		}
		close(jobs)
		wg.Wait()
	}() 

	return out
}
