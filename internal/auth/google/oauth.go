package google

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
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
			"https://www.googleapis.com/auth/calendar.events",
		},
		Endpoint: google.Endpoint,
	}
}

func CreateGmailService(ctx context.Context, refreshToken string) (*gmail.Service, error) {
	config := NewOAuthConfig()

	//Create token from stored refresh token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}

	tokenSource := config.TokenSource(ctx, token)
	httpClient := oauth2.NewClient(ctx, tokenSource)

	return gmail.NewService(ctx, option.WithHTTPClient(httpClient))
}

// CreateCalendarService creates a Google Calendar service using the stored refresh token
func CreateCalendarService(ctx context.Context, refreshToken string) (*calendar.Service, error) {
	config := NewOAuthConfig()

	token := &oauth2.Token{
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}

	tokenSource := config.TokenSource(ctx, token)
	httpClient := oauth2.NewClient(ctx, tokenSource)

	return calendar.NewService(ctx, option.WithHTTPClient(httpClient))
}
