package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

const JWTIssuer = "where-is-church"

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
			Issuer:    JWTIssuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// SetAuthCookie sets the JWT cookie with proper security flags.
func SetAuthCookie(c *gin.Context, token string, maxAge int, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", token, maxAge, "/", "", secure, true)
}

// ClearAuthCookie removes the JWT cookie.
func ClearAuthCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", "", -1, "/", "", secure, true)
}

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""

		if cookie, err := c.Cookie("token"); err == nil {
			tokenStr = cookie
		} else {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				tokenStr = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if tokenStr == "" {
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
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		}, jwt.WithIssuer(JWTIssuer))
		if err != nil || !token.Valid {
			ClearAuthCookie(c, false)
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
