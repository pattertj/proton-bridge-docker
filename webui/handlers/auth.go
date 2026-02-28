package handlers

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
)

// AuthMiddleware checks for a valid WEBUI_AUTH_TOKEN on API routes.
// If the env var is empty, auth is disabled (with a warning).
// Static frontend files are always served (the frontend handles the token prompt).
func AuthMiddleware(next http.Handler) http.Handler {
	token := os.Getenv("WEBUI_AUTH_TOKEN")
	if token == "" {
		log.Println("[auth] WARNING: WEBUI_AUTH_TOKEN is not set. Web UI has no authentication!")
		return next
	}
	log.Println("[auth] Authentication enabled via WEBUI_AUTH_TOKEN")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't require auth for static frontend files or the health endpoint
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header: "Bearer <token>"
		auth := r.Header.Get("Authorization")
		provided := strings.TrimPrefix(auth, "Bearer ")

		if auth == "" || provided == auth {
			// No Bearer prefix or no header at all
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
