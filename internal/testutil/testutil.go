package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/handlers"
	"github.com/wiseful-oak-systems/where-is-church/internal/middleware"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestApp holds all dependencies needed for integration tests.
type TestApp struct {
	DB     *gorm.DB
	Router *gin.Engine
	Cfg    *config.Config
	T      *testing.T
}

// NewTestApp creates a fully wired test application with an in-memory SQLite database.
func NewTestApp(t *testing.T) *TestApp {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Church{},
		&models.MassSchedule{},
		&models.CheckIn{},
		&models.Suggestion{},
		&models.Favorite{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	cfg := &config.Config{
		JWTSecret:    "test-secret-key-that-is-at-least-32-chars-long",
		JWTExpiry:    72,
		Port:         "8080",
		CookieSecure: false,
		Environment:  "development",
	}

	r := gin.New()

	authH := &handlers.AuthHandler{DB: db, Cfg: cfg}
	churchH := &handlers.ChurchHandler{DB: db}
	checkinH := &handlers.CheckInHandler{DB: db}
	suggestionH := &handlers.SuggestionHandler{DB: db}
	adminH := &handlers.AdminHandler{DB: db}
	favoriteH := &handlers.FavoriteHandler{DB: db}

	// Public API
	api := r.Group("/api")
	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)

	// Protected API
	auth := api.Group("/", middleware.AuthRequired(cfg.JWTSecret))
	auth.POST("/auth/logout", authH.Logout)
	auth.GET("/auth/me", authH.Me)
	auth.PUT("/auth/profile", authH.UpdateProfile)

	auth.GET("/churches", churchH.List)
	auth.GET("/churches/search", churchH.SearchNearby)
	auth.GET("/churches/:id", churchH.GetByID)
	auth.POST("/churches", churchH.Create)

	auth.POST("/checkins", checkinH.Create)
	auth.GET("/checkins/mine", checkinH.MyCheckIns)
	auth.GET("/checkins/stats", checkinH.UserStats)
	auth.GET("/churches/:id/checkins", checkinH.ChurchCheckIns)
	auth.GET("/churches/:id/loyal-users", checkinH.ChurchLoyalUsers)

	auth.POST("/suggestions", suggestionH.Create)
	auth.GET("/suggestions/mine", suggestionH.MySuggestions)

	auth.POST("/churches/:id/favorite", favoriteH.Add)
	auth.DELETE("/churches/:id/favorite", favoriteH.Remove)
	auth.GET("/churches/:id/favorite", favoriteH.Check)
	auth.GET("/favorites", favoriteH.List)

	mod := auth.Group("/", middleware.RoleRequired(models.RoleModerator, models.RoleAdmin))
	mod.PUT("/churches/:id", churchH.Update)
	mod.POST("/churches/:id/schedules", churchH.AddSchedule)
	mod.DELETE("/churches/:id/schedules/:scheduleId", churchH.DeleteSchedule)
	mod.GET("/suggestions", suggestionH.List)
	mod.PUT("/suggestions/:id", suggestionH.Review)

	adm := auth.Group("/admin", middleware.RoleRequired(models.RoleAdmin))
	adm.GET("/users", adminH.ListUsers)
	adm.PUT("/users/:id/role", adminH.SetRole)
	adm.PUT("/churches/:id/verify", adminH.VerifyChurch)
	adm.DELETE("/churches/:id", adminH.DeleteChurch)

	return &TestApp{DB: db, Router: r, Cfg: cfg, T: t}
}

// CreateUser registers a user and returns the JWT token.
func (app *TestApp) CreateUser(name, email, password, denomination string) string {
	app.T.Helper()
	body := map[string]string{
		"name": name, "email": email,
		"password": password, "denomination": denomination,
	}
	resp := app.Request("POST", "/api/auth/register", body, "")
	if resp.Code != http.StatusCreated {
		app.T.Fatalf("failed to create user %s: %d – %s", email, resp.Code, resp.Body.String())
	}
	var result map[string]any
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	return result["token"].(string)
}

// CreateAdmin registers a user and promotes them to admin.
func (app *TestApp) CreateAdmin(name, email, password string) string {
	app.T.Helper()
	app.CreateUser(name, email, password, "Catholic")
	app.DB.Model(&models.User{}).Where("email = ?", email).Update("role", models.RoleAdmin)
	// Re-login to get token with admin role
	resp := app.Request("POST", "/api/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	var result map[string]any
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	return result["token"].(string)
}

// CreateModerator registers a user and promotes them to moderator.
func (app *TestApp) CreateModerator(name, email, password string) string {
	app.T.Helper()
	app.CreateUser(name, email, password, "Catholic")
	app.DB.Model(&models.User{}).Where("email = ?", email).Update("role", models.RoleModerator)
	resp := app.Request("POST", "/api/auth/login", map[string]string{
		"email": email, "password": password,
	}, "")
	var result map[string]any
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	return result["token"].(string)
}

// SeedChurch creates a church directly in the database and returns its ID.
func (app *TestApp) SeedChurch(name, denomination, address string, lat, lng float64) uint {
	app.T.Helper()
	church := models.Church{
		Name: name, Denomination: denomination, Address: address,
		Latitude: lat, Longitude: lng, Verified: true, CreatedByID: 1,
	}
	if err := app.DB.Create(&church).Error; err != nil {
		app.T.Fatalf("failed to seed church: %v", err)
	}
	return church.ID
}

// Request performs an HTTP request against the test router.
func (app *TestApp) Request(method, path string, body any, token string) *httptest.ResponseRecorder {
	app.T.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	return w
}

// ParseJSON parses the response body into a map.
func ParseJSON(resp *httptest.ResponseRecorder) map[string]any {
	var result map[string]any
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	return result
}

// ParseJSONArray parses the response body into a slice of maps.
func ParseJSONArray(resp *httptest.ResponseRecorder) []map[string]any {
	var result []map[string]any
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	return result
}
