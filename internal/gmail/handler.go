package gmail

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/r7rainz/auramail/internal/auth"
	"github.com/r7rainz/auramail/internal/auth/google"
	"github.com/r7rainz/auramail/internal/user"
	"github.com/r7rainz/auramail/internal/utils"
)

type GmailHandler struct {
    userRepo user.Repository
}

func NewHandler(repo user.Repository) *GmailHandler {
    return &GmailHandler{userRepo: repo}
}

func (h *GmailHandler) SyncPlacementEmails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(auth.UserIDContextKey).(string)
	if !ok {
		http.Error(w, "Unauthorized: No UserID found", http.StatusUnauthorized)
	}

	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	srv, err := google.CreateGmailService(ctx, u.RefreshToken)
	if err != nil {
		log.Printf("Gmail Service Error: %v", err)
		http.Error(w, "Failed to connect to Gmail", 500)
		return
	}

	query := "from:placementoffice@vitbhopal.ac.in OR subject:placement"
	emails, err := utils.ListPlacementEmails(srv, query, 20)
	if err != nil {
		http.Error(w, "Extraction Failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Context-Type", "application/json")
	json.NewEncoder(w).Encode(emails)

}

func (h *GmailHandler) StreamPlacementEmails(w http.ResponseWriter, r *http.Request) {
	//setting headers to tell the browser to tell don't close the connection
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") //for development

	ctx := r.Context()

	rawID := ctx.Value(auth.UserIDContextKey)
	if rawID == nil {
		http.Error(w, "Unauthorized: No UserID", http.StatusUnauthorized)
	}

	userID := fmt.Sprintf("%v", rawID)
	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	srv, err := google.CreateGmailService(ctx, u.RefreshToken)
	if err != nil {
		http.Error(w, "Failed to initialize Gmail service", http.StatusUnauthorized)
		return
	}

	query := "from:placementoffice@vitbhopal.ac.in"
	emailStream := FetchAndSummarize(ctx, srv, query, u.ID)

	foundAny := false

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			//user closed the tab close everything
			return
		case summary, ok := <-emailStream:
			if !ok {
				//channel closed
				if !foundAny {
					fmt.Fprintf(w, "data: {\"error\": \"no_emails_found\"}\n\n")
				}
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
			//this is just a colon to keep the connection alive but doesn't trigger the JS logic
			fmt.Fprintf(w, ": heartbeat\n\n")
			w.(http.Flusher).Flush()
		}
	}

}
