package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
	"linkShortener/backend/config"
	"linkShortener/backend/models"
	"linkShortener/backend/routes"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	config.ConnectDB()
	config.DB.AutoMigrate(&models.User{}, &models.URL{}, &models.ClickAnalytics{})

	r := gin.Default()
	routes.SetupRoutes(r)

	r.Run(":" + os.Getenv("PORT"))
}
