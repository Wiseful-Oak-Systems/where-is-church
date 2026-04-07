package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PageHandler struct{}

func (h *PageHandler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"userID":       c.GetUint("userID"),
		"userRole":     c.GetString("userRole"),
		"denomination": c.GetString("userDenomination"),
	})
}

func (h *PageHandler) Login(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (h *PageHandler) RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func (h *PageHandler) ChurchDetail(c *gin.Context) {
	c.HTML(http.StatusOK, "church.html", gin.H{
		"churchID": c.Param("id"),
		"userID":   c.GetUint("userID"),
		"userRole": c.GetString("userRole"),
	})
}

func (h *PageHandler) Profile(c *gin.Context) {
	c.HTML(http.StatusOK, "profile.html", gin.H{
		"userID":   c.GetUint("userID"),
		"userRole": c.GetString("userRole"),
	})
}

func (h *PageHandler) AdminPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"userID":   c.GetUint("userID"),
		"userRole": c.GetString("userRole"),
	})
}
