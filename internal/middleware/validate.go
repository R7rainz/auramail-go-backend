package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type ValidationErrorResponse struct {
	Success bool         `json:"success"`
	Error   string       `json:"error"`
	Details []FieldError `json:"details"`
}

type FieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func ValidateBody[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body T
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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

			// safe type assertion
			if ve, ok := err.(validator.ValidationErrors); ok {
				for _, fe := range ve {
					details = append(details, FieldError{
						Path:    fe.StructNamespace(),
						Message: fe.Error(),
					})
				}
			} else {
				// fallback if err is some other type
				details = append(details, FieldError{
					Path:    "body",
					Message: err.Error(),
				})
			}

			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ValidationErrorResponse{
				Success: false,
				Error:   "Validation Error",
				Details: details,
			})
			return
		}

		// Pass validated body via context
		ctx := context.WithValue(r.Context(), "body", body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ValidateQuery[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		var query T

		b, _ := json.Marshal(r.URL.Query())
		if err := json.Unmarshal(b, &query); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ValidationErrorResponse{
				Success: false,
				Error: "Invalid Query",
				Details: []FieldError{{Path: "query", Message: err.Error()}},
			})
			return
		}
		if err := validate.Struct(query); err != nil {
			var details []FieldError
			if ve, ok := err.(validator.ValidationErrors); ok {
				for _, fe := range ve {
					details = append(details, FieldError{
						Path: fe.StructNamespace(),
						Message: fe.Error(),
					})
				}
			} else {
				details = append(details, FieldError{
					Path: "query", 
					Message: err.Error(),
				})
			}

			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ValidationErrorResponse{
				Success: false,
				Error: "Validation Error",
				Details: details,
			})
			return
		}
		ctx := context.WithValue(r.Context(), "query", query)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}


























