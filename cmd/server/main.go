package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/database"
	"github.com/wiseful-oak-systems/where-is-church/internal/handlers"
	"github.com/wiseful-oak-systems/where-is-church/internal/middleware"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

func main() {
	cfg := config.Load()
	db := database.Connect(cfg)

	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "web/static")

	// Handlers
	authH := &handlers.AuthHandler{DB: db, Cfg: cfg}
	churchH := &handlers.ChurchHandler{DB: db}
	checkinH := &handlers.CheckInHandler{DB: db}
	suggestionH := &handlers.SuggestionHandler{DB: db}
	adminH := &handlers.AdminHandler{DB: db}
	pageH := &handlers.PageHandler{}

	// Public pages
	r.GET("/login", pageH.Login)
	r.GET("/register", pageH.RegisterPage)

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

	// Moderator + Admin routes
	mod := auth.Group("/", middleware.RoleRequired(models.RoleModerator, models.RoleAdmin))
	mod.PUT("/churches/:id", churchH.Update)
	mod.POST("/churches/:id/schedules", churchH.AddSchedule)
	mod.DELETE("/churches/:id/schedules/:scheduleId", churchH.DeleteSchedule)
	mod.GET("/suggestions", suggestionH.List)
	mod.PUT("/suggestions/:id", suggestionH.Review)

	// Admin only routes
	adm := auth.Group("/admin", middleware.RoleRequired(models.RoleAdmin))
	adm.GET("/users", adminH.ListUsers)
	adm.PUT("/users/:id/role", adminH.SetRole)
	adm.PUT("/churches/:id/verify", adminH.VerifyChurch)
	adm.DELETE("/churches/:id", adminH.DeleteChurch)

	// Protected pages
	pages := r.Group("/", middleware.AuthRequired(cfg.JWTSecret))
	pages.GET("/", pageH.Index)
	pages.GET("/church/:id", pageH.ChurchDetail)
	pages.GET("/profile", pageH.Profile)
	pages.GET("/admin", middleware.RoleRequired(models.RoleAdmin), pageH.AdminPage)

	log.Printf("Server starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
