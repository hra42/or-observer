package handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// WithAuth returns middleware that checks for a valid Bearer token.
// If apiKey is empty, auth is disabled (backward compatible).
func WithAuth(apiKey string, next http.Handler) http.Handler {
	if apiKey == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return
		}

		next.ServeHTTP(w, r)
	})
}
