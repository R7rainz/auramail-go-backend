package google

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/r7rainz/auramail/internal/auth"
	"github.com/r7rainz/auramail/internal/user"
	"golang.org/x/oauth2"
)

type Handler struct {
	oauthConfig  *oauth2.Config
	userRepo     user.Repository
	stateManager *StateManager
	frontendURL  string
}

type GoogleUser struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewHandler(cfg *oauth2.Config, userRepo user.Repository, frontendURL string) *Handler {
	return &Handler{
		oauthConfig:  cfg,
		userRepo:     userRepo,
		stateManager: NewStateManager(),
		frontendURL:  frontendURL,
	}
}

func (h *Handler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	state, err := h.stateManager.Generate()
	if err != nil {
		slog.Error("failed to generate oauth state", "err", err)
		http.Error(w, "failed to start oauth flow", http.StatusInternalServerError)
		return
	}

	authURL := h.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !h.stateManager.Validate(state) {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	codeStr := r.URL.Query().Get("code")
	if codeStr == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	token, err := h.oauthConfig.Exchange(ctx, codeStr)
	if err != nil {
		slog.Error("oauth exchange failed", "err", err)
		http.Error(w, "oauth exchange failed", http.StatusInternalServerError)
		return
	}

	client := h.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		slog.Error("userinfo request failed", "err", err)
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
	if error := json.Unmarshal(body, &user); error != nil {
		http.Error(w, "invalid google user payload", http.StatusInternalServerError)
		return
	}

	slog.Info("google user authenticated", "email", user.Email, "sub", user.Sub)

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

	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	if token.RefreshToken != "" {
		//save the google token to db
		if err := h.userRepo.UpdateRefreshToken(ctx, u.ID, token.RefreshToken); err != nil {
			slog.Error("failed to save google refresh token", "err", err)
			http.Error(w, "failed to save token", http.StatusInternalServerError)
			return
		}
	} else {
		slog.Warn("no google refresh token received; using existing one")
	}

	accessToken, err := auth.GenerateAccessToken(
		u.ID,
		u.Email,
		u.Name,
	)
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	appRefreshToken, err := auth.GenerateRefreshToken(u.ID, u.Email)
	if err != nil {
		http.Error(w, "failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Redirect to frontend with tokens in URL
	redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s",
		h.frontendURL,
		url.QueryEscape(accessToken),
		url.QueryEscape(appRefreshToken),
	)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/auth/google", h.GoogleAuth)
	mux.HandleFunc("/auth/google/callback", h.GoogleCallback)
}
