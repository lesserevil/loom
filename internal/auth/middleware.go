package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// Middleware wraps an HTTP handler with authentication
func (m *Manager) Middleware(requiredPermission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Try API key auth
				apiKey := r.Header.Get("X-API-Key")
				if apiKey == "" {
					http.Error(w, "Missing authorization header", http.StatusUnauthorized)
					return
				}

				// Validate API key
				userID, permissions, err := m.ValidateAPIKey(apiKey)
				if err != nil {
					http.Error(w, "Invalid API key", http.StatusUnauthorized)
					return
				}

				// Check permission
				if requiredPermission != "" {
					hasPermission := false
					for _, p := range permissions {
						if p == requiredPermission || p == "*:*" {
							hasPermission = true
							break
						}
					}
					if !hasPermission {
						http.Error(w, "Insufficient permissions", http.StatusForbidden)
						return
					}
				}

				// Store userID in context
				r.Header.Set("X-User-ID", userID)
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from "Bearer <token>" format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := m.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
				return
			}

			// Check permission
			if requiredPermission != "" && !m.HasPermission(claims, requiredPermission) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			// Store claims in header for downstream handlers
			r.Header.Set("X-User-ID", claims.UserID)
			r.Header.Set("X-Username", claims.Username)
			r.Header.Set("X-Role", claims.Role)

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth wraps a handler with optional authentication
// (no error if auth fails, but stores claims if successful)
func (m *Manager) OptionalAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Try API key
				apiKey := r.Header.Get("X-API-Key")
				if apiKey != "" {
					if userID, _, err := m.ValidateAPIKey(apiKey); err == nil {
						r.Header.Set("X-User-ID", userID)
					}
				}
				next.ServeHTTP(w, r)
				return
			}

			// Extract token
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString := parts[1]
				if claims, err := m.ValidateToken(tokenString); err == nil {
					r.Header.Set("X-User-ID", claims.UserID)
					r.Header.Set("X-Username", claims.Username)
					r.Header.Set("X-Role", claims.Role)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromRequest extracts the user ID from request context
func GetUserIDFromRequest(r *http.Request) string {
	return r.Header.Get("X-User-ID")
}

// GetUsernameFromRequest extracts the username from request context
func GetUsernameFromRequest(r *http.Request) string {
	return r.Header.Get("X-Username")
}

// GetRoleFromRequest extracts the role from request context
func GetRoleFromRequest(r *http.Request) string {
	return r.Header.Get("X-Role")
}
