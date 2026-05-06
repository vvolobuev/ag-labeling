package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func JWTMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		tokenStr := strings.TrimPrefix(strings.TrimSpace(h), "Bearer ")
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Next()
	}
}

// OptionalJWTMiddleware sets userID from Bearer when token is valid; otherwise continues anonymously.
// Use for routes that allow both public guests and authenticated viewers (e.g. image files).
func OptionalJWTMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		tokenStr := strings.TrimPrefix(strings.TrimSpace(h), "Bearer ")
		if tokenStr == "" {
			c.Next()
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid || strings.TrimSpace(claims.UserID) == "" {
			c.Next()
			return
		}
		c.Set("userID", claims.UserID)
		c.Next()
	}
}

func UserID(c *gin.Context) string {
	v, ok := c.Get("userID")
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
