package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lovenest/backend/internal/models"
)

var JWTSecret = []byte("lovenest-ultra-secret-cozy-key-2026")

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	CoupleIDKey contextKey = "couple_id"
)

type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	CoupleID *int64 `json:"couple_id,omitempty"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int64, coupleID *int64) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		CoupleID: coupleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)), // 30 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Also check query param token for websockets
			authHeader = r.URL.Query().Get("token")
			if authHeader != "" {
				authHeader = "Bearer " + authHeader
			}
		}

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Unauthorized - Bearer token required",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &JWTClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Invalid or expired token",
			})
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		if claims.CoupleID != nil {
			ctx = context.WithValue(ctx, CoupleIDKey, *claims.CoupleID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(UserIDKey).(int64); ok {
		return v
	}
	return 0
}

func GetCoupleID(ctx context.Context) *int64 {
	if v, ok := ctx.Value(CoupleIDKey).(int64); ok {
		return &v
	}
	return nil
}

func AuthMiddlewareTokenParse(tokenStr string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

