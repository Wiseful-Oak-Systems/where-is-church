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

	store, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("storage initialization error: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())
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
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
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

	// Admin only routes (destructive operations)
	adm := auth.Group("/admin", middleware.RoleRequired(models.RoleAdmin))
	adm.DELETE("/churches/:id", adminH.DeleteChurch)

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
