package middleware

import (
	"crypto/subtle"
	"net/http"
)

func WebhookAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}

			providedSecret := r.Header.Get("X-Webhook-Secret")
			if providedSecret == "" {
				http.Error(w, "Missing X-Webhook-Secret header", http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare([]byte(secret), []byte(providedSecret)) != 1 {
				http.Error(w, "Invalid webhook secret", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}