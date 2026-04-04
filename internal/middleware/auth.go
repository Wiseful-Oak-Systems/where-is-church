package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

type Claims struct {
	UserID       uint        `json:"user_id"`
	Email        string      `json:"email"`
	Role         models.Role `json:"role"`
	Denomination string      `json:"denomination"`
	jwt.RegisteredClaims
}

func GenerateToken(user *models.User, secret string, expiryHours int) (string, error) {
	claims := &Claims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		Denomination: user.Denomination,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""

		// Check cookie first, then Authorization header
		if cookie, err := c.Cookie("token"); err == nil {
			tokenStr = cookie
		} else {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				tokenStr = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if tokenStr == "" {
			// For API calls, return 401. For page requests, redirect to login.
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			} else {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
			}
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.SetCookie("token", "", -1, "/", "", false, true)
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			} else {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
			}
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", string(claims.Role))
		c.Set("userDenomination", claims.Denomination)
		c.Next()
	}
}

func RoleRequired(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		userRole := models.Role(role.(string))
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
