package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User table
type User struct {
	ID           string    `gorm:"primaryKey;type:text"`
	Username     string    `gorm:"size:32;not null;uniqueIndex"`
	PasswordHash string    `gorm:"type:text;not null"`
	Email        string    `gorm:"size:255;not null;uniqueIndex"`
	IsAdmin      bool      `gorm:"not null;default:false"`
	IsActive     bool      `gorm:"not null;default:true"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"not null;autoUpdateTime"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return
}

// Session table
type Session struct {
	ID         string    `gorm:"primaryKey;type:text"`
	Data       string    `gorm:"not null"`
	ExpiryDate time.Time `gorm:"index;not null"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return
}
