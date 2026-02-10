package auth

import (
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/r7rainz/auramail/internal/user"
)

type Handler struct {
	oauthConfig *oauth2.Config
	userRepo    user.Repository
	service     *Service
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type userResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
}

func NewHandler(cfg *oauth2.Config, userRepo user.Repository) *Handler {
	return &Handler{
		oauthConfig: cfg,
		userRepo:    userRepo,
		service:     NewService(userRepo),
	}
}

// Me returns the current authenticated user
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(int)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "unauthorized",
		})
		return
	}

	foundUser, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "user not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user": userResponse{
			ID:    foundUser.ID,
			Email: foundUser.Email,
			Name:  foundUser.Name,
		},
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid request",
		})
		return
	}

	_, err := ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid refresh token",
		})
		return
	}

	foundUser, err := h.userRepo.FindByRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid refresh token",
		})
		return
	}
	_ = foundUser

	accessToken, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid refresh token",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"accessToken": accessToken,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(int)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "unauthorized",
		})
		return
	}

	if err := h.service.Logout(r.Context(), userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "logout failed",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
