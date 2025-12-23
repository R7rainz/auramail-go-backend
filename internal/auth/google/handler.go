package google

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/r7rainz/auramail/internal/auth"
	"github.com/r7rainz/auramail/internal/user"
	"golang.org/x/oauth2"
)

type Handler struct {
	oauthConfig *oauth2.Config
	userRepo    user.Repository
}

type GoogleUser struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewHandler(cfg *oauth2.Config) *Handler {
	return &Handler{oauthConfig: cfg}
}

func (h *Handler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	state := "random-state-for-now"

	authURL := h.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	codeStr := r.URL.Query().Get("code")
	if codeStr == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	token, err := h.oauthConfig.Exchange(ctx, codeStr)
	if err != nil {
		log.Printf("exchange failed: %v", err)
		http.Error(w, "oauth exchange failed", http.StatusInternalServerError)
		return
	}

	client := h.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		log.Printf("userinfo request failed: %v", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "invalid google response", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read response", http.StatusInternalServerError)
		return
	}

	var user GoogleUser
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(w, "invalid google user payload", http.StatusInternalServerError)
		return
	}

	log.Printf("google user: %s (%s)", user.Email, user.Sub)

	u, err := h.userRepo.FindOrCreateGoogleUser(
		ctx,
		user.Email,
		user.Name,
		user.Sub,
	)
	if err != nil {
		http.Error(w, "failed to persist user", http.StatusInternalServerError)
		return
	}

	accessToken, err := auth.GenerateAccessToken(
		u.ID,
		u.Email,
		u.Name,
	)

	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"accessToken": accessToken,
	})
}
