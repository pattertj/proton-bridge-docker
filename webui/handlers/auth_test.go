package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})
}

func TestAuthMiddleware_NoToken_PassesThrough(t *testing.T) {
	os.Unsetenv("WEBUI_AUTH_TOKEN")
	handler := AuthMiddleware(newTestHandler())

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 when no token configured, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	os.Setenv("WEBUI_AUTH_TOKEN", "test-secret")
	defer os.Unsetenv("WEBUI_AUTH_TOKEN")
	handler := AuthMiddleware(newTestHandler())

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 with valid token, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	os.Setenv("WEBUI_AUTH_TOKEN", "test-secret")
	defer os.Unsetenv("WEBUI_AUTH_TOKEN")
	handler := AuthMiddleware(newTestHandler())

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 with invalid token, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid token" {
		t.Errorf("expected 'invalid token' error, got %q", resp["error"])
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	os.Setenv("WEBUI_AUTH_TOKEN", "test-secret")
	defer os.Unsetenv("WEBUI_AUTH_TOKEN")
	handler := AuthMiddleware(newTestHandler())

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 with no header, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "authentication required" {
		t.Errorf("expected 'authentication required', got %q", resp["error"])
	}
}

func TestAuthMiddleware_NoBearerPrefix(t *testing.T) {
	os.Setenv("WEBUI_AUTH_TOKEN", "test-secret")
	defer os.Unsetenv("WEBUI_AUTH_TOKEN")
	handler := AuthMiddleware(newTestHandler())

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "test-secret") // no Bearer prefix
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 without Bearer prefix, got %d", w.Code)
	}
}

func TestAuthMiddleware_HealthBypass(t *testing.T) {
	os.Setenv("WEBUI_AUTH_TOKEN", "test-secret")
	defer os.Unsetenv("WEBUI_AUTH_TOKEN")
	handler := AuthMiddleware(newTestHandler())

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 for /api/health without auth, got %d", w.Code)
	}
}

func TestAuthMiddleware_StaticFilesBypass(t *testing.T) {
	os.Setenv("WEBUI_AUTH_TOKEN", "test-secret")
	defer os.Unsetenv("WEBUI_AUTH_TOKEN")
	handler := AuthMiddleware(newTestHandler())

	for _, path := range []string{"/", "/index.html", "/styles.css", "/app.js"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("expected 200 for %s without auth, got %d", path, w.Code)
		}
	}
}
