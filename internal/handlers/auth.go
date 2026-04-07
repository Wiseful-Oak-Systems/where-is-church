package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/middleware"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

type RegisterInput struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	Denomination string `json:"denomination"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new account with name, email, password, and optional denomination (defaults to Catholic)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterInput  true  "Registration data"
// @Success      201   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len([]byte(input.Password)) > models.MaxBcryptPasswordLen {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("password must not exceed %d bytes", models.MaxBcryptPasswordLen)})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Denomination == "" {
		input.Denomination = "Catholic"
	}

	var existing models.User
	if err := h.DB.WithContext(c.Request.Context()).Where("email = ?", input.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	user := models.User{
		Name:         input.Name,
		Email:        input.Email,
		Denomination: input.Denomination,
		Role:         models.RoleUser,
	}
	if err := user.SetPassword(input.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := h.DB.WithContext(c.Request.Context()).Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token, err := middleware.GenerateToken(&user, h.Cfg.JWTSecret, h.Cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	middleware.SetAuthCookie(c, token, h.Cfg.JWTExpiry*3600, h.Cfg.CookieSecure)
	c.JSON(http.StatusCreated, gin.H{
		"user":  user,
		"token": token,
	})
}

// Login godoc
// @Summary      Authenticate user
// @Description  Login with email and password, returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      LoginInput  true  "Login credentials"
// @Success      200   {object}  map[string]any
// @Failure      401   {object}  map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	var user models.User
	if err := h.DB.WithContext(c.Request.Context()).Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !user.CheckPassword(input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := middleware.GenerateToken(&user, h.Cfg.JWTSecret, h.Cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	middleware.SetAuthCookie(c, token, h.Cfg.JWTExpiry*3600, h.Cfg.CookieSecure)
	c.JSON(http.StatusOK, gin.H{
		"user":  user,
		"token": token,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	middleware.ClearAuthCookie(c, h.Cfg.CookieSecure)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Me godoc
// @Summary      Get current user profile
// @Description  Returns the authenticated user's profile data
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.User
// @Failure      401  {object}  map[string]string
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetUint("userID")
	var user models.User
	if err := h.DB.WithContext(c.Request.Context()).First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update name, denomination, and/or location coordinates
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.User
// @Failure      400  {object}  map[string]string
// @Router       /auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		Name         string   `json:"name"`
		Denomination string   `json:"denomination"`
		Latitude     *float64 `json:"latitude"`
		Longitude    *float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Denomination != "" {
		updates["denomination"] = input.Denomination
	}
	if input.Latitude != nil {
		if *input.Latitude < -90 || *input.Latitude > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "latitude must be between -90 and 90"})
			return
		}
		updates["latitude"] = *input.Latitude
	}
	if input.Longitude != nil {
		if *input.Longitude < -180 || *input.Longitude > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "longitude must be between -180 and 180"})
			return
		}
		updates["longitude"] = *input.Longitude
	}

	ctx := c.Request.Context()
	if err := h.DB.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	var user models.User
	if err := h.DB.WithContext(ctx).First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated profile"})
		return
	}
	c.JSON(http.StatusOK, user)
}
