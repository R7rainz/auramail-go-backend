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

type RefreshTokenClaims struct {
	UserID int `json:"sub"`
	Email string `json:"email"`
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

func ValidateAccessToken(token string) (*AccessTokenClaims, error) {
	key, err := jwtSecret()
	if err != nil {
		return nil, err 
	}

	parsedToken, err := jwt.ParseWithClaims(
		token,
		&AccessTokenClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return key, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := parsedToken.Claims.(*AccessTokenClaims)
	if !ok || !parsedToken.Valid {
		return nil, errors.New("invalid access token claims")
	}

	return claims, nil
}

func ValidateRefreshToken(token string) (*RefreshTokenClaims, error) {
	key, err := jwtSecret()
	if err != nil {
		return nil, err 
	}

	parsedToken, err := jwt.ParseWithClaims(
		token,
		&RefreshTokenClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return key, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := parsedToken.Claims.(*RefreshTokenClaims)
	if !ok || !parsedToken.Valid {
		return nil, errors.New("invalid refresh token claims")
	}

	return claims, nil
}

func GenerateRefreshToken(userID int, email string) (string, error) {
	expirationTime := time.Now().Add(14 * 24 * time.Hour)
	claims := &RefreshTokenClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "AuraMail",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	key, err := jwtSecret()
	if err!= nil {
		return "", err
	}

	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("coult not sign the token %w", err)
	}
	return tokenString, nil
}
