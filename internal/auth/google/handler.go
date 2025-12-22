package google

import (
	"net/http"

	"golang.org/x/oauth2"
)

type Handler struct {
	oauthConfig *oauth2.Config
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

	token, err := h.oauthConfig.Exchange(ctx,codeStr)
	if err != nil {
		http.Error(w, "failed to exchange code", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("google auth success"))
	 
	_ = token
}
