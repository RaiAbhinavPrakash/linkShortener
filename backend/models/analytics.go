package models

import (
	"gorm.io/gorm"
)

type ClickAnalytics struct {
	gorm.Model
	URLID     uint   `gorm:"not null"` // Foreign Key
	IPAddress string
	Referrer  string
	UserAgent string
}
