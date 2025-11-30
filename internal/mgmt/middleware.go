package mgmt

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// withAuth wraps a handler with authentication middleware
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if no token is configured
		if s.token == "" {
			next(w, r)
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Check Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(auth, bearerPrefix) {
			writeJSONError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		token := strings.TrimPrefix(auth, bearerPrefix)
		if token != s.token {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		// Token valid, proceed
		next(w, r)
	}
}

// withLogging wraps a handler with request logging
func (s *Server) withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		slog.Debug("API request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
		)

		// Call next handler
		next(w, r)

		// Log response time
		duration := time.Since(start)
		slog.Debug("API response",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", duration.String(),
		)
	}
}
