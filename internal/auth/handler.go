package auth

import (
	"encoding/json"
	"net/http"

	"github.com/r7rainz/auramail/internal/user"
	"golang.org/x/oauth2"
)

type Handler struct {
	oauthConfig *oauth2.Config
	userRepo     user.Repository
	service     *Service
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func NewHandler(cfg *oauth2.Config, userRepo user.Repository) *Handler {
	return &Handler {
		oauthConfig: cfg, 
		userRepo: userRepo,
		service: NewService(userRepo),
	}
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	_, err  := ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	foundUser, err := h.userRepo.FindByRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	_ = foundUser

	accessToken, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Context-Type", "application/json")
	if err:= json.NewEncoder(w).Encode(map[string]string{
		"access_token": accessToken,
	}); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.service.Logout(r.Context(), userID); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := r.Context().Value("userID").(int)
// 	if !ok {
// 		http.Error(w, "unauthorized", http.StatusUnauthorized)
// 		return
// 	}
//
// 	if err := h.service.Logout(r.Context(), userID); err != nil {
// 		http.Error(w, "logout failed", http.StatusInternalServerError)
// 		return
// 	}
//
// 	w.WriteHeader(http.StatusOK)
// }
