package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/database"
	"github.com/wiseful-oak-systems/where-is-church/internal/handlers"
	"github.com/wiseful-oak-systems/where-is-church/internal/i18n"
	"github.com/wiseful-oak-systems/where-is-church/internal/seeder"
	"github.com/wiseful-oak-systems/where-is-church/internal/middleware"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/storage"
)

func main() {
	cfg := config.Load()

	if cfg.IsProd() {
		if err := cfg.Validate(); err != nil {
			log.Fatalf("configuration error: %v", err)
		}
		gin.SetMode(gin.ReleaseMode)
	}

	db := database.Connect(cfg)

	// Seed default tags (always, idempotent)
	seeder.SeedDefaultTags(db)

	// Auto-seed churches from OpenStreetMap if database is empty
	if cfg.AutoSeed {
		seeder.SeedIfEmpty(db, cfg.SeedCountry)
	}

	store, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("storage initialization error: %v", err)
	}

	if _, err := i18n.Load("locales"); err != nil {
		log.Fatalf("i18n initialization error: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(i18n.Middleware())
	if !cfg.IsProd() {
		r.Use(gin.Logger())
	}

	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "web/static")
	// Serve local uploads if using local storage
	if cfg.StorageDriver == "local" {
		r.Static("/uploads", cfg.StorageLocalPath)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Handlers
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
	attachmentH := &handlers.AttachmentHandler{
		DB:      db,
		Store:   store,
		MaxSize: int64(cfg.MaxUploadSizeMB) * 1024 * 1024,
	}
	pageH := &handlers.PageHandler{}

	// Public pages
	r.GET("/login", pageH.Login)
	r.GET("/register", pageH.RegisterPage)

	// Custom JSON 404/405
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(i18n.GetLang(c), "not_found")})
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": i18n.T(i18n.GetLang(c), "method_not_allowed")})
	})

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

	// Church ownership claims (any authenticated user can claim)
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

	// Moderator, Church Owner, Community Manager, and Admin routes
	mod := auth.Group("/", middleware.RoleRequired(models.RoleModerator, models.RoleChurchOwner, models.RoleCommunityManager, models.RoleAdmin))
	mod.PUT("/churches/:id", churchH.Update)
	mod.POST("/churches/:id/schedules", churchH.AddSchedule)
	mod.DELETE("/churches/:id/schedules/:scheduleId", churchH.DeleteSchedule)
	mod.GET("/suggestions", suggestionH.List)
	mod.PUT("/suggestions/:id", suggestionH.Review)

	// Community Manager + Admin routes
	cmgr := auth.Group("/admin", middleware.RoleRequired(models.RoleCommunityManager, models.RoleAdmin))
	cmgr.GET("/users", adminH.ListUsers)
	cmgr.PUT("/users/:id/role", adminH.SetRole)
	cmgr.GET("/moderators", adminH.ListModerators)
	cmgr.GET("/claims", ownershipH.ListClaims)
	cmgr.PUT("/claims/:id", ownershipH.ReviewClaim)
	cmgr.PUT("/churches/:id/verify", adminH.VerifyChurch)

	// Admin tools (CM + Admin)
	cmgr.GET("/audit-log", adminToolsH.GetAuditLog)
	cmgr.GET("/data-health", adminToolsH.DataHealthDashboard)
	cmgr.GET("/duplicates", adminToolsH.FindDuplicateChurches)
	cmgr.GET("/stale-churches", adminToolsH.StaleChurches)
	cmgr.POST("/bulk/verify-churches", adminToolsH.BulkVerifyChurches)
	cmgr.POST("/bulk/review-suggestions", adminToolsH.BulkReviewSuggestions)

	// Admin only routes (destructive operations)
	adm := auth.Group("/admin", middleware.RoleRequired(models.RoleAdmin))
	adm.DELETE("/churches/:id", adminH.DeleteChurch)
	adm.POST("/merge-churches", adminToolsH.MergeChurches)

	// Protected pages
	pages := r.Group("/", middleware.AuthRequired(cfg.JWTSecret))
	pages.GET("/", pageH.Index)
	pages.GET("/church/:id", pageH.ChurchDetail)
	pages.GET("/profile", pageH.Profile)
	pages.GET("/admin", middleware.RoleRequired(models.RoleCommunityManager, models.RoleAdmin), pageH.AdminPage)

	// Graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("Server stopped gracefully")
}
