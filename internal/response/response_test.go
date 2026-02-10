package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{"message": "hello"}
	JSON(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %s", contentType)
	}

	var result map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("expected 'hello', got '%s'", result["message"])
	}
}

func TestSuccess(t *testing.T) {
	rr := httptest.NewRecorder()

	Success(rr, map[string]string{"key": "value"})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var result APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestCreated(t *testing.T) {
	rr := httptest.NewRecorder()

	Created(rr, map[string]int{"id": 1})

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}

	var result APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Success {
		t.Error("expected success to be true")
	}
}

func TestNoContent(t *testing.T) {
	rr := httptest.NewRecorder()

	NoContent(rr)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestError(t *testing.T) {
	rr := httptest.NewRecorder()

	Error(rr, http.StatusBadRequest, ErrCodeBadRequest, "Invalid input", map[string]string{"field": "email"})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var result APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Success {
		t.Error("expected success to be false")
	}

	if result.Error == nil {
		t.Fatal("expected error to be present")
	}

	if result.Error.Code != ErrCodeBadRequest {
		t.Errorf("expected code 'bad_request', got '%s'", result.Error.Code)
	}

	if result.Error.Message != "Invalid input" {
		t.Errorf("expected message 'Invalid input', got '%s'", result.Error.Message)
	}
}

func TestUnauthorized(t *testing.T) {
	rr := httptest.NewRecorder()

	Unauthorized(rr, "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}

	var result APIResponse
	json.NewDecoder(rr.Body).Decode(&result)

	if result.Error.Code != ErrCodeUnauthorized {
		t.Errorf("expected code 'unauthorized', got '%s'", result.Error.Code)
	}

	if result.Error.Message != "Authentication required" {
		t.Errorf("expected default message, got '%s'", result.Error.Message)
	}
}

func TestNotFound(t *testing.T) {
	rr := httptest.NewRecorder()

	NotFound(rr, "User not found")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}

	var result APIResponse
	json.NewDecoder(rr.Body).Decode(&result)

	if result.Error.Message != "User not found" {
		t.Errorf("expected 'User not found', got '%s'", result.Error.Message)
	}
}

func TestRateLimited(t *testing.T) {
	rr := httptest.NewRecorder()

	RateLimited(rr, "60")

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rr.Code)
	}

	retryAfter := rr.Header().Get("Retry-After")
	if retryAfter != "60" {
		t.Errorf("expected Retry-After '60', got '%s'", retryAfter)
	}
}

func TestSuccessWithMeta(t *testing.T) {
	rr := httptest.NewRecorder()

	data := []string{"item1", "item2"}
	meta := &Meta{
		Page:       1,
		PerPage:    10,
		Total:      100,
		TotalPages: 10,
	}

	SuccessWithMeta(rr, data, meta)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var result APIResponse
	json.NewDecoder(rr.Body).Decode(&result)

	if result.Meta == nil {
		t.Fatal("expected meta to be present")
	}

	if result.Meta.Total != 100 {
		t.Errorf("expected total 100, got %d", result.Meta.Total)
	}
}
