package middleware

import (
	"context"
	"net/http"
	"strings"

	authv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/auth/v1"
)

type contextKey string

const UserIDCtxKey = contextKey("userID")

// AuthMiddleware extracts the JWT from the Authorization header and verifies it via Auth gRPC Service
func AuthMiddleware(authClient authv1.AuthServiceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				next.ServeHTTP(w, r)
				return
			}

			token := parts[1]
			resp, err := authClient.ValidateToken(r.Context(), &authv1.ValidateTokenRequest{
				Token: token,
			})
			if err != nil || resp == nil || !resp.GetIsValid() {
				next.ServeHTTP(w, r)
				return
			}

			// Inject user ID into context
			ctx := context.WithValue(r.Context(), UserIDCtxKey, resp.GetUserId())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext retrieves the userID from context if present
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDCtxKey).(string)
	return userID, ok
}
