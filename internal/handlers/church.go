package handlers

import (
	"math"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

const (
	DefaultSearchRadiusKm = 10.0
	MaxSearchRadiusKm     = 100.0
	DefaultPageLimit      = 50
	MaxPageLimit          = 200
)

var timeFormatRe = regexp.MustCompile(`^\d{2}:\d{2}$`)

type ChurchHandler struct {
	DB *gorm.DB
}

func (h *ChurchHandler) SearchNearby(c *gin.Context) {
	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil || lat < -90 || lat > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat must be a valid latitude (-90 to 90)"})
		return
	}
	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng must be a valid longitude (-180 to 180)"})
		return
	}

	radiusKm := DefaultSearchRadiusKm
	if r := c.Query("radius"); r != "" {
		if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 {
			radiusKm = math.Min(parsed, MaxSearchRadiusKm)
		}
	}

	denomination := c.Query("denomination")
	if denomination == "" {
		if d, exists := c.Get("userDenomination"); exists {
			denomination = d.(string)
		}
	}

	haversine := `(6371 * acos(
		cos(radians(?)) * cos(radians(latitude)) *
		cos(radians(longitude) - radians(?)) +
		sin(radians(?)) * sin(radians(latitude))
	))`

	ctx := c.Request.Context()
	query := h.DB.WithContext(ctx).Model(&models.Church{}).
		Select("*, "+haversine+" AS distance", lat, lng, lat).
		Where(haversine+" <= ?", lat, lng, lat, radiusKm).
		Order("distance ASC")

	if denomination != "" && denomination != "All" {
		query = query.Where("denomination = ?", denomination)
	}

	var churches []struct {
		models.Church
		Distance float64 `json:"distance"`
	}

	if err := query.Preload("Schedules").Find(&churches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, churches)
}

func (h *ChurchHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	var church models.Church
	if err := h.DB.WithContext(c.Request.Context()).Preload("Schedules").Preload("CreatedBy").First(&church, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}
	c.JSON(http.StatusOK, church)
}

type CreateChurchInput struct {
	Name         string  `json:"name" binding:"required,max=200"`
	Denomination string  `json:"denomination"`
	Address      string  `json:"address" binding:"required,max=500"`
	Latitude     float64 `json:"latitude" binding:"required"`
	Longitude    float64 `json:"longitude" binding:"required"`
	Phone        string  `json:"phone" binding:"max=50"`
	Website      string  `json:"website" binding:"max=500"`
	Description  string  `json:"description" binding:"max=2000"`
}

func (h *ChurchHandler) Create(c *gin.Context) {
	var input CreateChurchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Latitude < -90 || input.Latitude > 90 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude must be between -90 and 90"})
		return
	}
	if input.Longitude < -180 || input.Longitude > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "longitude must be between -180 and 180"})
		return
	}

	userID := c.GetUint("userID")
	if input.Denomination == "" {
		input.Denomination = "Catholic"
	}

	church := models.Church{
		Name:         input.Name,
		Denomination: input.Denomination,
		Address:      input.Address,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		Phone:        input.Phone,
		Website:      input.Website,
		Description:  input.Description,
		CreatedByID:  userID,
	}

	role := c.GetString("userRole")
	if role == string(models.RoleAdmin) || role == string(models.RoleModerator) {
		church.Verified = true
		now := time.Now()
		church.LastVerified = &now
	}

	if err := h.DB.WithContext(c.Request.Context()).Create(&church).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create church"})
		return
	}

	c.JSON(http.StatusCreated, church)
}

type UpdateChurchInput struct {
	Name         string  `json:"name" binding:"max=200"`
	Denomination string  `json:"denomination"`
	Address      string  `json:"address" binding:"max=500"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Phone        string  `json:"phone" binding:"max=50"`
	Website      string  `json:"website" binding:"max=500"`
	Description  string  `json:"description" binding:"max=2000"`
}

func (h *ChurchHandler) Update(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var church models.Church
	if err := h.DB.WithContext(ctx).First(&church, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	var input UpdateChurchInput
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
	if input.Address != "" {
		updates["address"] = input.Address
	}
	if input.Latitude != 0 {
		if input.Latitude < -90 || input.Latitude > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "latitude must be between -90 and 90"})
			return
		}
		updates["latitude"] = input.Latitude
	}
	if input.Longitude != 0 {
		if input.Longitude < -180 || input.Longitude > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "longitude must be between -180 and 180"})
			return
		}
		updates["longitude"] = input.Longitude
	}
	if input.Phone != "" {
		updates["phone"] = input.Phone
	}
	if input.Website != "" {
		updates["website"] = input.Website
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}

	if err := h.DB.WithContext(ctx).Model(&church).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update church"})
		return
	}

	if err := h.DB.WithContext(ctx).Preload("Schedules").First(&church, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated church"})
		return
	}
	c.JSON(http.StatusOK, church)
}

func (h *ChurchHandler) AddSchedule(c *gin.Context) {
	churchID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid church id"})
		return
	}

	ctx := c.Request.Context()
	var church models.Church
	if err := h.DB.WithContext(ctx).First(&church, churchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	var input struct {
		Type      string `json:"type"`
		DayOfWeek int    `json:"day_of_week" binding:"min=0,max=6"`
		StartTime string `json:"start_time" binding:"required"`
		EndTime   string `json:"end_time"`
		Language  string `json:"language"`
		Notes     string `json:"notes" binding:"max=500"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !timeFormatRe.MatchString(input.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time must be in HH:MM format"})
		return
	}
	if input.EndTime != "" && !timeFormatRe.MatchString(input.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_time must be in HH:MM format"})
		return
	}

	schedType := models.ScheduleType(input.Type)
	if schedType == "" {
		schedType = models.ScheduleMass
	}
	if !models.ValidScheduleTypes[schedType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: mass, confession, adoration"})
		return
	}

	if input.Language == "" {
		input.Language = "English"
	}

	schedule := models.MassSchedule{
		ChurchID:  uint(churchID),
		Type:      schedType,
		DayOfWeek: input.DayOfWeek,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
		Language:  input.Language,
		Notes:     input.Notes,
	}

	if err := h.DB.WithContext(ctx).Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add schedule"})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

func (h *ChurchHandler) DeleteSchedule(c *gin.Context) {
	churchID := c.Param("id")
	scheduleID := c.Param("scheduleId")

	ctx := c.Request.Context()
	result := h.DB.WithContext(ctx).Where("id = ? AND church_id = ?", scheduleID, churchID).Delete(&models.MassSchedule{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found for this church"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "schedule deleted"})
}

func (h *ChurchHandler) List(c *gin.Context) {
	var churches []models.Church
	ctx := c.Request.Context()
	query := h.DB.WithContext(ctx).Preload("Schedules").Order("name ASC")

	if denom := c.Query("denomination"); denom != "" && denom != "All" {
		query = query.Where("denomination = ?", denom)
	}

	limit := parseLimit(c.Query("limit"), DefaultPageLimit)
	offset := parseOffset(c.Query("offset"))

	if err := query.Limit(limit).Offset(offset).Find(&churches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list churches"})
		return
	}
	c.JSON(http.StatusOK, churches)
}

func parseLimit(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultVal
	}
	if v > MaxPageLimit {
		return MaxPageLimit
	}
	return v
}

func parseOffset(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
