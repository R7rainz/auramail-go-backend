package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessTokenClaims struct {
	UserID int    `json:"sub"`
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
	jwt.RegisteredClaims
}

func jwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET not set")
	}
	return []byte(secret), nil
}

func GenerateAccessToken(userID int, email, name string) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &AccessTokenClaims{
		UserID: userID,
		Email:  email,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "AuraMail",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	key, err := jwtSecret()
	if err != nil {
		return "", err
	}
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("could not sign the token : %w", err)
	}

	return tokenString, nil
}

func ValidateAccessToken(token string) (*AccessTokenClaims, error)
