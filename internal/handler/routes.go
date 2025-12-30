package handler

import (
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/labstack/echo/v4"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, authHandler *AuthHandler, profileHandler *ProfileHandler) {
	// API version 1
	api := e.Group("/api/v1")

	// Auth routes (protected)
	auth := api.Group("/auth")
	auth.Use(authMiddleware.Authenticate())
	auth.POST("/callback", authHandler.Callback)
	auth.GET("/me", authHandler.Me)
	auth.POST("/logout", authHandler.Logout)

	// Profile routes (protected)
	profile := api.Group("/profile")
	profile.Use(authMiddleware.Authenticate())
	profile.GET("", profileHandler.GetProfile)
	profile.PUT("", profileHandler.UpdateProfile)
}
