package controllers

import (
	"encoding/csv"
	"fmt"
	"linkShortener/backend/config"
	"linkShortener/backend/models"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateShortCode(length int) string {
	rand.Seed(time.Now().UnixNano())
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func ShortenURL(c *gin.Context) {
	var input struct {
		OriginalURL string `json:"original_url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.OriginalURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	shortCode := generateShortCode(6)
	url := models.URL{
		OriginalURL: input.OriginalURL,
		ShortCode:   shortCode,
		UserID:      userID.(uint),
	}
	config.DB.Create(&url)

	c.JSON(http.StatusOK, gin.H{
		"short_url":    "/s/" + shortCode,
		"original_url": url.OriginalURL,
	})
}

func RedirectURL(c *gin.Context) {
	code := c.Param("code")
	log.Println("Received redirect for code:", code)

	var url models.URL
	result := config.DB.Where("short_code = ?", code).First(&url)
	if result.Error != nil {
		log.Println("URL not found in DB:", result.Error)
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// Log success
	log.Println("Redirecting to:", url.OriginalURL)

	// Analytics logging
	click := models.ClickAnalytics{
		URLID:     url.ID,
		IPAddress: c.ClientIP(),
		Referrer:  c.Request.Referer(),
		UserAgent: c.Request.UserAgent(),
	}
	config.DB.Create(&click)

	// Increment click count
	config.DB.Model(&url).Update("click_count", url.ClickCount+1)

	c.Redirect(http.StatusFound, url.OriginalURL) // Changed to 302 for safety
}

// func GetUserLinks(c *gin.Context) {
// 	userID := c.MustGet("user_id").(uint)

// 	var links []models.URL
// 	config.DB.Where("user_id = ?", userID).Find(&links)

// 	c.JSON(http.StatusOK, links)
// }

func GetUserLinks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse pagination params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	var urls []models.URL
	query := config.DB.Where("user_id = ?", userID)

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("original_url ILIKE ? OR short_code ILIKE ?", searchTerm, searchTerm)
	}

	var total int64
	query.Model(&models.URL{}).Count(&total)

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&urls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch links"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       urls,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": int((total + int64(limit) - 1) / int64(limit)),
	})
}

func GetLinkAnalytics(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	code := c.Param("code")

	var url models.URL
	if err := config.DB.Where("short_code = ? AND user_id = ?", code, userID).First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
		return
	}

	// --- Pagination ---
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Total count of analytics for the URL
	var total int64
	config.DB.Model(&models.ClickAnalytics{}).Where("url_id = ?", url.ID).Count(&total)

	// Fetch paginated analytics logs
	var logs []models.ClickAnalytics
	config.DB.Where("url_id = ?", url.ID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&logs)

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"short_code":   url.ShortCode,
		"original_url": url.OriginalURL,
		"click_count":  url.ClickCount,
		"analytics":    logs,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

func UpdateShortURL(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var url models.URL
	if err := config.DB.First(&url, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	if url.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to update this URL"})
		return
	}

	var input struct {
		OriginalURL string `json:"original_url" binding:"required,url"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	url.OriginalURL = input.OriginalURL
	config.DB.Save(&url)

	c.JSON(http.StatusOK, gin.H{"message": "URL updated successfully", "data": url})
}

func DeleteShortURL(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var url models.URL
	if err := config.DB.First(&url, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	if url.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to delete this URL"})
		return
	}

	// Delete associated analytics
	config.DB.Where("url_id = ?", url.ID).Delete(&models.ClickAnalytics{})
	// Delete the URL itself
	config.DB.Delete(&url)

	c.JSON(http.StatusOK, gin.H{"message": "URL deleted successfully"})
}

func ExportAnalyticsCSV(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	code := c.Param("code")

	var url models.URL
	if err := config.DB.Where("short_code = ? AND user_id = ?", code, userID).First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
		return
	}

	var logs []models.ClickAnalytics
	config.DB.Where("url_id = ?", url.ID).Order("created_at desc").Find(&logs)

	// Set CSV headers
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=analytics_%s_%d.csv", code, time.Now().Unix()))
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write CSV headers
	writer.Write([]string{"ID", "URL ID", "IP Address", "Referrer", "User Agent", "Created At"})

	// Write each record
	for _, log := range logs {
		writer.Write([]string{
			strconv.Itoa(int(log.ID)),
			strconv.Itoa(int(log.URLID)),
			log.IPAddress,
			log.Referrer,
			log.UserAgent,
			log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
}
