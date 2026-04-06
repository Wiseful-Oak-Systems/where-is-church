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
	"github.com/wiseful-oak-systems/where-is-church/internal/i18n"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/storage"
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

	allModels := []any{
		&models.User{},
		&models.Church{},
		&models.MassSchedule{},
		&models.CheckIn{},
		&models.Suggestion{},
		&models.Favorite{},
		&models.ChurchOwnership{},
		&models.Attachment{},
		&models.AuditLog{},
		&models.UserReputation{},
		&models.ChurchConfirmation{},
		&models.ChurchProposal{},
		&models.Tag{},
		&models.ChurchTag{},
	}
	for _, m := range allModels {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatalf("failed to migrate %T: %v", m, err)
		}
	}

	cfg := &config.Config{
		JWTSecret:    "test-secret-key-that-is-at-least-32-chars-long",
		JWTExpiry:    72,
		Port:         "8080",
		CookieSecure: false,
		Environment:  "development",
	}

	// Load minimal i18n translations for tests (English only)
	i18n.LoadFromMap(map[string]map[string]string{
		"en-US": {
			"auth.required":              "authentication required",
			"auth.invalid_token":         "invalid token",
			"auth.invalid_credentials":   "invalid credentials",
			"auth.forbidden":             "forbidden",
			"auth.insufficient_permissions": "insufficient permissions",
			"not_found":                  "not found",
			"method_not_allowed":         "method not allowed",
		},
		"pt-BR": {
			"auth.required":              "authentication required",
			"auth.invalid_token":         "invalid token",
			"auth.invalid_credentials":   "invalid credentials",
			"auth.forbidden":             "forbidden",
			"auth.insufficient_permissions": "insufficient permissions",
			"not_found":                  "not found",
			"method_not_allowed":         "method not allowed",
		},
	})

	r := gin.New()
	r.Use(i18n.Middleware())

	authH := &handlers.AuthHandler{DB: db, Cfg: cfg}
	churchH := &handlers.ChurchHandler{DB: db}
	checkinH := &handlers.CheckInHandler{DB: db}
	suggestionH := &handlers.SuggestionHandler{DB: db}
	adminH := &handlers.AdminHandler{DB: db}
	favoriteH := &handlers.FavoriteHandler{DB: db}
	ownershipH := &handlers.OwnershipHandler{DB: db}
	adminToolsH := &handlers.AdminToolsHandler{DB: db}
	reputationH := &handlers.ReputationHandler{DB: db}
	communityH := &handlers.CommunityHandler{DB: db}
	tagH := &handlers.TagHandler{DB: db}
	testStore, _ := storage.NewLocalStore(t.TempDir())
	attachmentH := &handlers.AttachmentHandler{DB: db, Store: testStore, MaxSize: 10 * 1024 * 1024}

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

	auth.POST("/churches/:id/claim", ownershipH.ClaimChurch)
	auth.GET("/my-churches", ownershipH.MyChurches)
	auth.GET("/my-churches/claims", ownershipH.MyClaims)

	auth.GET("/reputation", reputationH.GetMyReputation)
	auth.GET("/impact", communityH.ContributionImpact)
	auth.POST("/churches/:id/confirm", communityH.ConfirmChurch)

	auth.GET("/tags", tagH.ListTags)
	auth.GET("/churches/:id/tags", tagH.GetChurchTags)
	auth.POST("/churches/:id/tags", tagH.AddTagToChurch)
	auth.GET("/churches/by-tag", tagH.SearchByTag)
	auth.GET("/leaderboard", reputationH.Leaderboard)

	auth.POST("/attachments", attachmentH.Upload)
	auth.GET("/attachments", attachmentH.List)
	auth.GET("/attachments/:id/download", attachmentH.Download)
	auth.DELETE("/attachments/:id", attachmentH.Delete)

	mod := auth.Group("/", middleware.RoleRequired(models.RoleModerator, models.RoleChurchOwner, models.RoleCommunityManager, models.RoleAdmin))
	mod.PUT("/churches/:id", churchH.Update)
	mod.POST("/churches/:id/schedules", churchH.AddSchedule)
	mod.DELETE("/churches/:id/schedules/:scheduleId", churchH.DeleteSchedule)
	mod.GET("/suggestions", suggestionH.List)
	mod.PUT("/suggestions/:id", suggestionH.Review)

	cmgr := auth.Group("/admin", middleware.RoleRequired(models.RoleCommunityManager, models.RoleAdmin))
	cmgr.GET("/users", adminH.ListUsers)
	cmgr.PUT("/users/:id/role", adminH.SetRole)
	cmgr.GET("/moderators", adminH.ListModerators)
	cmgr.GET("/claims", ownershipH.ListClaims)
	cmgr.PUT("/claims/:id", ownershipH.ReviewClaim)
	cmgr.PUT("/churches/:id/verify", adminH.VerifyChurch)
	cmgr.GET("/audit-log", adminToolsH.GetAuditLog)
	cmgr.GET("/data-health", adminToolsH.DataHealthDashboard)
	cmgr.GET("/duplicates", adminToolsH.FindDuplicateChurches)
	cmgr.GET("/stale-churches", adminToolsH.StaleChurches)
	cmgr.POST("/bulk/verify-churches", adminToolsH.BulkVerifyChurches)
	cmgr.POST("/bulk/review-suggestions", adminToolsH.BulkReviewSuggestions)

	adm := auth.Group("/admin", middleware.RoleRequired(models.RoleAdmin))
	adm.DELETE("/churches/:id", adminH.DeleteChurch)
	adm.POST("/merge-churches", adminToolsH.MergeChurches)

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

// CreateCommunityManager registers a user and promotes them to community_manager.
func (app *TestApp) CreateCommunityManager(name, email, password string) string {
	app.T.Helper()
	app.CreateUser(name, email, password, "Catholic")
	app.DB.Model(&models.User{}).Where("email = ?", email).Update("role", models.RoleCommunityManager)
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
		Latitude: lat, Longitude: lng, Verified: true,
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
