package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidationErrorResponse is the standard error format for validation failures
type ValidationErrorResponse struct {
	Success bool         `json:"success"`
	Error   string       `json:"error"`
	Details []FieldError `json:"details"`
}

// FieldError describes a single validation error
type FieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidateBody creates middleware that validates JSON request bodies
func ValidateBody[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body T
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ValidationErrorResponse{
				Success: false,
				Error:   "Invalid JSON",
				Details: []FieldError{{Path: "body", Message: err.Error()}},
			})
			return
		}

		if err := validate.Struct(body); err != nil {
			var details []FieldError

			// Safe type assertion
			if ve, ok := err.(validator.ValidationErrors); ok {
				for _, fe := range ve {
					details = append(details, FieldError{
						Path:    fe.StructNamespace(),
						Message: fe.Error(),
					})
				}
			} else {
				// Fallback if err is some other type
				details = append(details, FieldError{
					Path:    "body",
					Message: err.Error(),
				})
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ValidationErrorResponse{
				Success: false,
				Error:   "Validation Error",
				Details: details,
			})
			return
		}

		// Pass validated body via context
		ctx := context.WithValue(r.Context(), validatedBodyKey, body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateQuery creates middleware that validates query parameters
func ValidateQuery[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var query T

		b, _ := json.Marshal(r.URL.Query())
		if err := json.Unmarshal(b, &query); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ValidationErrorResponse{
				Success: false,
				Error:   "Invalid Query",
				Details: []FieldError{{Path: "query", Message: err.Error()}},
			})
			return
		}

		if err := validate.Struct(query); err != nil {
			var details []FieldError
			if ve, ok := err.(validator.ValidationErrors); ok {
				for _, fe := range ve {
					details = append(details, FieldError{
						Path:    fe.StructNamespace(),
						Message: fe.Error(),
					})
				}
			} else {
				details = append(details, FieldError{
					Path:    "query",
					Message: err.Error(),
				})
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ValidationErrorResponse{
				Success: false,
				Error:   "Validation Error",
				Details: details,
			})
			return
		}

		ctx := context.WithValue(r.Context(), validatedQueryKey, query)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Context keys for validated data
type contextKey string

const (
	validatedBodyKey  contextKey = "validatedBody"
	validatedQueryKey contextKey = "validatedQuery"
)

// GetValidatedBody retrieves the validated body from context
func GetValidatedBody[T any](ctx context.Context) (T, bool) {
	val, ok := ctx.Value(validatedBodyKey).(T)
	return val, ok
}

// GetValidatedQuery retrieves the validated query from context
func GetValidatedQuery[T any](ctx context.Context) (T, bool) {
	val, ok := ctx.Value(validatedQueryKey).(T)
	return val, ok
}
