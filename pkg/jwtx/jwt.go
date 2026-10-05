package jwtx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/planly/pkg/httpx"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrMissingToken = errors.New("authorization token is required")
)

// Claims represents the JWT claims for authenticated users.
type Claims struct {
	UserID uuid.UUID `json:"user_id,omitempty"`
	jwt.RegisteredClaims
}

// SignToken generates a signed JWT token string for a user.
func SignToken(userID uuid.UUID, secret []byte, duration time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}

// VerifyToken verifies and parses the JWT token string using the secret.
func VerifyToken(tokenStr string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Verify UserID is populated; fallback to parsing Subject
	if claims.UserID == uuid.Nil && claims.Subject != "" {
		parsed, err := uuid.Parse(claims.Subject)
		if err == nil {
			claims.UserID = parsed
		}
	}

	return claims, nil
}

// WithUserID stores the user ID into the context.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDContextKey, userID)
}

// GetUserID retrieves the user ID from the context.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return val, ok
}

// AuthMiddleware validates the JWT from Authorization header and injects the user ID into context.
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
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

			claims, err := VerifyToken(parts[1], secret)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
				return
			}

			ctx := WithUserID(r.Context(), claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireInternalKey middleware checks X-Internal-Key header for service-to-service communication.
func RequireInternalKey(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Internal-Key")
			if key == "" || key != expectedKey {
				httpx.WriteError(w, http.StatusForbidden, "forbidden", "Invalid internal service key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
