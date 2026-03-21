// Package middleware provides HTTP middleware for the GoodClass API.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Context keys used by handlers to retrieve authenticated user info.
const (
	ContextKeyUserID   = "auth_user_id"
	ContextKeyNickname = "auth_nickname"
	ContextKeyIsAdmin  = "auth_is_admin"
)

var jwtSecret []byte

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Generate a random secret if not set (will invalidate tokens on restart).
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("failed to generate JWT secret: " + err.Error())
		}
		secret = hex.EncodeToString(b)
	}
	jwtSecret = []byte(secret)
}

// Claims defines the JWT payload.
type Claims struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	IsAdmin  bool   `json:"is_admin,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user.
// Token expires in 30 days.
func GenerateToken(userID, nickname string, isAdmin bool) (string, error) {
	claims := Claims{
		UserID:   userID,
		Nickname: nickname,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gclass",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken validates and parses a JWT string.
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// AuthOptional is a Gin middleware that parses a JWT from the Authorization
// header if present, but does NOT reject requests without a token.
// Use on public routes that optionally personalize responses (e.g. is_own).
func AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.Next()
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			c.Next()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyNickname, claims.Nickname)
		c.Set(ContextKeyIsAdmin, claims.IsAdmin)
		c.Next()
	}
}

// AuthRequired is a Gin middleware that requires a valid JWT in the
// Authorization header (Bearer <token>).
// On success it sets ContextKeyUserID and ContextKeyNickname in the context.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyNickname, claims.Nickname)
		c.Set(ContextKeyIsAdmin, claims.IsAdmin)
		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from the Gin context.
// Returns empty string if not authenticated.
func GetUserID(c *gin.Context) string {
	v, _ := c.Get(ContextKeyUserID)
	s, _ := v.(string)
	return s
}

// GetNickname extracts the authenticated nickname from the Gin context.
func GetNickname(c *gin.Context) string {
	v, _ := c.Get(ContextKeyNickname)
	s, _ := v.(string)
	return s
}

func GetIsAdmin(c *gin.Context) bool {
	v, _ := c.Get(ContextKeyIsAdmin)
	b, _ := v.(bool)
	return b
}
