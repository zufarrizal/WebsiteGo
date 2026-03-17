package main

import (
	"log"

	"websitego/internal/config"
	"websitego/internal/database"
	"websitego/internal/handlers"
	"websitego/internal/middleware"
	"websitego/internal/migrations"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewMySQL(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := migrations.Run(cfg, db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	router := gin.Default()
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}
	router.Use(middleware.SilenceClientAbortErrors())
	router.LoadHTMLGlob("templates/*")
	router.Static("/assets", "./assets")
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})

	store := cookie.NewStore([]byte(cfg.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24,
		HttpOnly: true,
		SameSite: cfg.CookieSameSite(),
		Secure:   cfg.CookieSecure,
	})
	router.Use(sessions.Sessions("websitego_session", store))

	dashboardHandler := handlers.NewDashboardHandler(db)
	authHandler := handlers.NewAuthHandler(db, dashboardHandler)
	if err := dashboardHandler.WarmupTotalUsers(); err != nil {
		log.Printf("warning: failed to warmup total users cache: %v", err)
	}

	router.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("user_id") != nil {
			c.Redirect(302, "/dashboard")
			return
		}
		c.Redirect(302, "/login")
	})

	guest := router.Group("/")
	guest.Use(middleware.RequireGuest())
	guest.GET("/register", authHandler.ShowRegister)
	guest.POST("/register", authHandler.Register)
	guest.GET("/login", authHandler.ShowLogin)
	guest.POST("/login", authHandler.Login)

	router.GET("/logout", authHandler.Logout)

	protected := router.Group("/")
	protected.Use(middleware.RequireAuth())
	protected.GET("/dashboard", dashboardHandler.Index)
	protected.POST("/dashboard/users", dashboardHandler.CreateUser)
	protected.POST("/dashboard/users/:id/update", dashboardHandler.UpdateUser)
	protected.POST("/dashboard/users/:id/delete", dashboardHandler.DeleteUser)

	log.Printf("server running on :%s", cfg.AppPort)
	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
