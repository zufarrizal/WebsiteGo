package handlers

import (
	"net/http"
	"strings"

	"websitego/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db        *gorm.DB
	dashboard *DashboardHandler
}

func NewAuthHandler(db *gorm.DB, dashboard *DashboardHandler) *AuthHandler {
	return &AuthHandler{db: db, dashboard: dashboard}
}

func (h *AuthHandler) ShowRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"title": "Register",
		"error": "",
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")

	if name == "" || email == "" || len(password) < 6 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title": "Register",
			"error": "Nama, email, dan password minimal 6 karakter wajib diisi.",
		})
		return
	}

	var existing models.User
	if err := h.db.Where("email = ?", email).First(&existing).Error; err == nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"title": "Register",
			"error": "Email sudah terdaftar.",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"title": "Register",
			"error": "Gagal memproses password.",
		})
		return
	}

	role := "user"
	var totalUsers int64
	if err := h.db.Model(&models.User{}).Count(&totalUsers).Error; err == nil && totalUsers == 0 {
		role = "admin"
	}

	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := h.db.Create(&user).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"title": "Register",
			"error": "Gagal menyimpan user.",
		})
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":   "Login",
		"error":   "",
		"success": "Registrasi berhasil. Silakan login.",
	})
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":   "Login",
		"error":   "",
		"success": "",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	password := c.PostForm("password")

	var user models.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Login",
			"error": "Email atau password salah.",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"title": "Login",
			"error": "Email atau password salah.",
		})
		return
	}

	if user.Role != "admin" {
		var adminCount int64
		if err := h.db.Model(&models.User{}).Where("role = ?", "admin").Count(&adminCount).Error; err == nil && adminCount == 0 {
			user.Role = "admin"
			_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Update("role", "admin").Error
		}
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("user_name", user.Name)
	session.Set("user_role", user.Role)
	if err := session.Save(); err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"title": "Login",
			"error": "Gagal membuat session.",
		})
		return
	}

	// Render dashboard directly to avoid redirect and extra request.
	if h.dashboard != nil {
		h.dashboard.renderDashboardWithSuccess(c, "", "Login berhasil.")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":   "Login",
		"error":   "",
		"success": "Login berhasil.",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	_ = session.Save()
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":   "Login",
		"error":   "",
		"success": "Logout berhasil.",
	})
}
