package middleware

import (
	"net/http"
	"strings"

	"github.com/planly/pkg/httpx"
	"github.com/planly/pkg/jwtx"
)

// JWTAuthMiddleware verifies incoming JWT, injects X-User-ID, and forwards the Authorization header.
func JWTAuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid authorization header format")
				return
			}

			claims, err := jwtx.VerifyToken(parts[1], secret)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
				return
			}

			// Forward X-User-ID header to downstream services
			r.Header.Set("X-User-ID", claims.UserID.String())

			// Keep context enriched with UserID
			ctx := jwtx.WithUserID(r.Context(), claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
