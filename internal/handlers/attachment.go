package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/storage"
	"gorm.io/gorm"
)

type AttachmentHandler struct {
	DB      *gorm.DB
	Store   storage.Store
	MaxSize int64 // max upload size in bytes
}

// Upload godoc
// @Summary      Upload a file attachment
// @Description  Upload a document or image and link it to a suggestion, claim, or church.
//
//	Accepted formats: JPEG, PNG, WebP, GIF, PDF. Max size configurable via MAX_UPLOAD_SIZE_MB.
//
// @Tags         attachments
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file             formData  file    true   "File to upload"
// @Param        attachable_type  formData  string  true   "Entity type: suggestion, claim, or church"
// @Param        attachable_id    formData  int     true   "Entity ID"
// @Success      201  {object}  models.Attachment
// @Failure      400  {object}  map[string]string
// @Failure      413  {object}  map[string]string
// @Router       /attachments [post]
func (h *AttachmentHandler) Upload(c *gin.Context) {
	userID := c.GetUint("userID")
	ctx := c.Request.Context()

	attachableType := models.AttachableType(c.PostForm("attachable_type"))
	if attachableType != models.AttachSuggestion && attachableType != models.AttachClaim && attachableType != models.AttachChurch {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attachable_type must be suggestion, claim, or church"})
		return
	}

	attachableID, err := strconv.ParseUint(c.PostForm("attachable_id"), 10, 32)
	if err != nil || attachableID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attachable_id must be a positive integer"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > h.MaxSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("file too large (max %d MB)", h.MaxSize/(1024*1024)),
		})
		return
	}

	// Detect and validate content type
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process file"})
		return
	}

	if !storage.AllowedDocTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file type not allowed (accepted: JPEG, PNG, WebP, GIF, PDF)"})
		return
	}

	// Generate storage key and save
	prefix := string(attachableType) + "s"
	key := storage.GenerateKey(prefix, header.Filename)

	savedKey, err := h.Store.Save(ctx, key, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store file"})
		return
	}

	attachment := models.Attachment{
		UploadedByID:   userID,
		AttachableType: attachableType,
		AttachableID:   uint(attachableID),
		Filename:       header.Filename,
		StorageKey:     savedKey,
		ContentType:    contentType,
		SizeBytes:      header.Size,
	}

	if err := h.DB.WithContext(ctx).Create(&attachment).Error; err != nil {
		// Clean up stored file on DB error
		_ = h.Store.Delete(ctx, savedKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save attachment record"})
		return
	}

	attachment.URL = h.Store.URL(savedKey)
	c.JSON(http.StatusCreated, attachment)
}

// List godoc
// @Summary      List attachments for an entity
// @Description  Returns all attachments linked to a specific suggestion, claim, or church.
// @Tags         attachments
// @Produce      json
// @Security     BearerAuth
// @Param        attachable_type  query  string  true  "Entity type"
// @Param        attachable_id    query  int     true  "Entity ID"
// @Success      200  {array}  models.Attachment
// @Router       /attachments [get]
func (h *AttachmentHandler) List(c *gin.Context) {
	attachableType := c.Query("attachable_type")
	attachableID := c.Query("attachable_id")
	ctx := c.Request.Context()

	if attachableType == "" || attachableID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "attachable_type and attachable_id are required"})
		return
	}

	var attachments []models.Attachment
	if err := h.DB.WithContext(ctx).
		Where("attachable_type = ? AND attachable_id = ?", attachableType, attachableID).
		Order("created_at DESC").
		Find(&attachments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list attachments"})
		return
	}

	for i := range attachments {
		attachments[i].URL = h.Store.URL(attachments[i].StorageKey)
	}

	c.JSON(http.StatusOK, attachments)
}

// Download godoc
// @Summary      Download an attachment
// @Description  Stream the file content for a specific attachment.
// @Tags         attachments
// @Produce      octet-stream
// @Security     BearerAuth
// @Param        id  path  int  true  "Attachment ID"
// @Success      200
// @Router       /attachments/{id}/download [get]
func (h *AttachmentHandler) Download(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var attachment models.Attachment
	if err := h.DB.WithContext(ctx).First(&attachment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}

	reader, err := h.Store.Get(ctx, attachment.StorageKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve file"})
		return
	}
	defer func() { _ = reader.Close() }()

	c.Header("Content-Type", attachment.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, attachment.Filename))
	if _, err := io.Copy(c.Writer, reader); err != nil {
		c.Status(http.StatusInternalServerError)
	}
}

// Delete godoc
// @Summary      Delete an attachment
// @Description  Remove an attachment. Only the uploader, moderators, or admins can delete.
// @Tags         attachments
// @Security     BearerAuth
// @Param        id  path  int  true  "Attachment ID"
// @Success      200  {object}  map[string]string
// @Router       /attachments/{id} [delete]
func (h *AttachmentHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetUint("userID")
	userRole := models.Role(c.GetString("userRole"))
	ctx := c.Request.Context()

	var attachment models.Attachment
	if err := h.DB.WithContext(ctx).First(&attachment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}

	// Only uploader or moderator+ can delete
	if attachment.UploadedByID != userID && !models.HasAtLeastRole(userRole, models.RoleModerator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you can only delete your own attachments"})
		return
	}

	if err := h.Store.Delete(ctx, attachment.StorageKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}

	if err := h.DB.WithContext(ctx).Delete(&attachment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete attachment record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "attachment deleted"})
}
