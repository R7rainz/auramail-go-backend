package google

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func NewOAuthConfig() *oauth2.Config {
	redirectURL := os.Getenv("GOOGLE_OAUTH_REDIRECT_URI")
	if redirectURL == "" {
		fmt.Println("Warning: GOOGLE_OAUTH_REDIRECT_URI is not set.")
	}

	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:  redirectURL,

		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/gmail.readonly",
		},
		Endpoint: google.Endpoint,
	}
}
