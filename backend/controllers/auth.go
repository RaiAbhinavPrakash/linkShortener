package controllers

import (
	"linkShortener/backend/config"
	"linkShortener/backend/middleware"
	"linkShortener/backend/models"
	"linkShortener/backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Register(c *gin.Context) {
	var input struct {
		Username string
		Email    string
		Password string
	}
	c.BindJSON(&input)

	hashedPassword, _ := utils.HashPassword(input.Password)

	user := models.User{
		Username: input.Username,
		Email:    input.Email,
		Password: hashedPassword,
	}
	config.DB.Create(&user)

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully. Please login."})
}

func Login(c *gin.Context) {
	var input struct {
		Email    string
		Password string
	}
	c.BindJSON(&input)

	var user models.User
	result := config.DB.Where("email = ?", input.Email).First(&user)
	if result.Error != nil || !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token, _ := utils.GenerateToken(user.ID)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func DeleteUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Step 1: Fetch user's URLs
	var urls []models.URL
	config.DB.Where("user_id = ?", user.ID).Find(&urls)

	// Step 2: Delete all analytics linked to these URLs
	for _, url := range urls {
		config.DB.Unscoped().Where("url_id = ?", url.ID).Delete(&models.ClickAnalytics{})
	}

	// Step 3: Delete URLs
	config.DB.Unscoped().Where("user_id = ?", user.ID).Delete(&models.URL{})

	// Step 4: Delete the user permanently
	config.DB.Unscoped().Delete(&user)

	// Step 5: Invalidate token
	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	middleware.AddTokenToBlacklist(tokenString)

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var updateData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if updateData.Username != "" {
		user.Username = updateData.Username
	}
	if updateData.Email != "" {
		user.Email = updateData.Email
	}

	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Profile updated successfully",
		"username": user.Username,
		"email":    user.Email,
	})
}
