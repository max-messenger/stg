package middlewares

import (
	"context"
	"fmt"
	"net/http"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func GetUserID(ctx context.Context) int64 {
	uid, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		return 0
	}

	return uid
}

type ParseTokenFunc func(ctx context.Context, token string) (userID int64, err error)

func AuthMiddleware(parseToken ParseTokenFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing authorization header")

				return
			}

			userID, err := parseToken(r.Context(), token)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid access token")

				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := fmt.Fprintf(w, `{"error":%q}`, msg); err != nil {
		return
	}
}
