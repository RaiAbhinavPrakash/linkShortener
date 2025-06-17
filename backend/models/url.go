package models

import (
	"gorm.io/gorm"
)

type URL struct {
	gorm.Model
	OriginalURL string `gorm:"not null"`
	ShortCode   string `gorm:"uniqueIndex;not null"`
	UserID      uint   `gorm:"not null"`
	ClickCount  uint
}
