// Package middleware provides HTTP middleware components for authentication, logging, and request processing.
package middleware

import (
	"net/http"
)

func AuthMiddleware(authKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check x-ins-auth-key header first
			key := r.Header.Get("x-ins-auth-key")

			// If x-ins-auth-key is not present, check Authorization header
			if key == "" {
				key = r.Header.Get("Authorization")
			}

			if key != authKey {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
