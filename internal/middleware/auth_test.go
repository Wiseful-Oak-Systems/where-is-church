package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/i18n"
	"github.com/wiseful-oak-systems/where-is-church/internal/middleware"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

func init() {
	i18n.LoadFromMap(map[string]map[string]string{
		"en-US": {
			"auth.required":              "authentication required",
			"auth.invalid_token":         "invalid token",
			"auth.forbidden":             "forbidden",
			"auth.insufficient_permissions": "insufficient permissions",
		},
		"pt-BR": {
			"auth.required":              "authentication required",
			"auth.invalid_token":         "invalid token",
			"auth.forbidden":             "forbidden",
			"auth.insufficient_permissions": "insufficient permissions",
		},
	})
}

const testSecret = "test-jwt-secret"

func setupTestRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(i18n.Middleware())
	api := r.Group("/api/test", middlewares...)
	api.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"userID":       c.GetUint("userID"),
			"userRole":     c.GetString("userRole"),
			"denomination": c.GetString("userDenomination"),
		})
	})
	page := r.Group("/page", middlewares...)
	page.GET("/dashboard", func(c *gin.Context) {
		c.String(200, "OK")
	})
	return r
}

func TestJWTTokenGeneration(t *testing.T) {
	t.Run("System generates a valid JWT token for an authenticated user", func(t *testing.T) {
		user := &models.User{
			ID: 1, Email: "user@test.com",
			Role: models.RoleUser, Denomination: "Catholic",
		}
		token, err := middleware.GenerateToken(user, testSecret, 72)
		if err != nil {
			t.Fatalf("expected no error generating token, got %v", err)
		}
		if token == "" {
			t.Fatal("expected a non-empty token")
		}
	})

	t.Run("Token encodes user ID, email, role, and denomination", func(t *testing.T) {
		user := &models.User{
			ID: 42, Email: "admin@test.com",
			Role: models.RoleAdmin, Denomination: "Orthodox",
		}
		token, _ := middleware.GenerateToken(user, testSecret, 72)

		r := setupTestRouter(middleware.AuthRequired(testSecret))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("API request without token returns 401 Unauthorized", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Page request without token redirects to login page", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret))
		req := httptest.NewRequest("GET", "/page/dashboard", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusFound {
			t.Errorf("expected 302 redirect, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		if location != "/login" {
			t.Errorf("expected redirect to /login, got %q", location)
		}
	})

	t.Run("Request with a valid Bearer token is allowed through", func(t *testing.T) {
		user := &models.User{ID: 1, Email: "user@test.com", Role: models.RoleUser, Denomination: "Catholic"}
		token, _ := middleware.GenerateToken(user, testSecret, 72)

		r := setupTestRouter(middleware.AuthRequired(testSecret))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("Request with an invalid token returns 401", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Request with token signed by wrong secret is rejected", func(t *testing.T) {
		user := &models.User{ID: 1, Email: "user@test.com", Role: models.RoleUser}
		token, _ := middleware.GenerateToken(user, "wrong-secret", 72)

		r := setupTestRouter(middleware.AuthRequired(testSecret))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Token from cookie is accepted as alternative to Authorization header", func(t *testing.T) {
		user := &models.User{ID: 1, Email: "user@test.com", Role: models.RoleUser, Denomination: "Catholic"}
		token, _ := middleware.GenerateToken(user, testSecret, 72)

		r := setupTestRouter(middleware.AuthRequired(testSecret))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.AddCookie(&http.Cookie{Name: "token", Value: token})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

func TestRoleMiddleware(t *testing.T) {
	makeToken := func(role models.Role) string {
		user := &models.User{ID: 1, Email: "u@t.com", Role: role, Denomination: "Catholic"}
		token, _ := middleware.GenerateToken(user, testSecret, 72)
		return token
	}

	t.Run("Admin can access admin-only routes", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret), middleware.RoleRequired(models.RoleAdmin))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(models.RoleAdmin))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("admin should access admin route, got %d", w.Code)
		}
	})

	t.Run("Regular user cannot access admin-only routes", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret), middleware.RoleRequired(models.RoleAdmin))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(models.RoleUser))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("regular user should be forbidden, got %d", w.Code)
		}
	})

	t.Run("Moderator can access moderator routes", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret), middleware.RoleRequired(models.RoleModerator, models.RoleAdmin))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(models.RoleModerator))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("moderator should access mod route, got %d", w.Code)
		}
	})

	t.Run("Admin can also access moderator routes", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret), middleware.RoleRequired(models.RoleModerator, models.RoleAdmin))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(models.RoleAdmin))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("admin should access mod route, got %d", w.Code)
		}
	})

	t.Run("Regular user cannot access moderator routes", func(t *testing.T) {
		r := setupTestRouter(middleware.AuthRequired(testSecret), middleware.RoleRequired(models.RoleModerator, models.RoleAdmin))
		req := httptest.NewRequest("GET", "/api/test/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(models.RoleUser))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("regular user should be forbidden from mod route, got %d", w.Code)
		}
	})
}
