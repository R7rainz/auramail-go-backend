package google

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

type Handler struct {
	oauthConfig *oauth2.Config
}

type GoogleUser struct {
	Sub  string `json:"sub"`
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
		log.Printf("error message: %v", err)
		http.Error(w, "error message", http.StatusInternalServerError)
		return
	}


	_ = token

	client := h.oauthConfig.Client(ctx, token)
	response, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		log.Printf("userinfo request failed : %v", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()
	
	if response.StatusCode != http.StatusOK{
		http.Error(w, "invalid google response", http.StatusUnauthorized)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "failed to read response", http.StatusInternalServerError)
	}

	var user GoogleUser
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(w, "invalid google user payload", http.StatusInternalServerError)
		return
	}

	log.Printf("google user : %s (%s)", user.Email, user.Sub)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("google auth success"))
}
