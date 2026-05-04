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

// Project table
type Status string

const (
	StatusCreating Status = "creating" // todo: needs to revist and make the statuses proper.
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
)

type Project struct {
	ID           string     `gorm:"primaryKey;type:text"`
	Name         string     `gorm:"size:255;not null;uniqueIndex"`
	Domain       string     `gorm:"size:255;not null;uniqueIndex"`
	Tag          string     `gorm:"type:text;not null"`
	Description  string     `gorm:"type:text"`
	Port         string     `gorm:"size:16;not null"`
	Image        string     `gorm:"type:text"`
	Status       Status     `gorm:"type:text;not null;default:creating"`
	IsActive     bool       `gorm:"not null;default:true"`
	LastActiveAt *time.Time `gorm:"index"`
	ContainerID  string     `gorm:"size:255"`
	CreatedAt    time.Time  `gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"not null;autoUpdateTime"`
	CreatedByID  string     `gorm:"type:text;not null;index"`
	CreatedBy    User       `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	UpdatedByID  string     `gorm:"type:text;not null;index"`
	UpdatedBy    User       `gorm:"foreignKey:UpdatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return
}

func (s Status) Valid() bool {
	return s == StatusCreating || s == StatusRunning || s == StatusStopped
}
