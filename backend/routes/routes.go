package routes

import (
	"linkShortener/backend/controllers"
	"linkShortener/backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
	}

	api := r.Group("/api")
	api.Use(middleware.JWTAuthMiddleware())
	{
		api.POST("/shorten", controllers.ShortenURL)
		api.GET("/links", controllers.GetUserLinks)
		api.GET("/links/:code/analytics", controllers.GetLinkAnalytics)
		api.DELETE("/user", controllers.DeleteUser)
		api.PUT("/links/:id", controllers.UpdateShortURL)
		api.DELETE("/links/:id", controllers.DeleteShortURL)
		api.PUT("/user/update", controllers.UpdateUserProfile)
	}

	r.GET("/s/:code", controllers.RedirectURL)
	r.HEAD("/s/:code", controllers.RedirectURL)
}
