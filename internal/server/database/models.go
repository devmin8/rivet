package database

import (
	"fmt"
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
	StatusStopped   Status = "stopped"
	StatusDeploying Status = "deploying"
	StatusRunning   Status = "running"
	StatusFailed    Status = "failed"
)

type DesiredStatus string

const (
	DesiredStatusRunning DesiredStatus = "running"
	DesiredStatusStopped DesiredStatus = "stopped"
)

type Platform string

const (
	PlatformLinuxAMD64 Platform = "linux/amd64"
	PlatformLinuxARM64 Platform = "linux/arm64"
)

type Project struct {
	ID          string `gorm:"primaryKey;type:text"`
	Name        string `gorm:"size:255;not null;uniqueIndex"`
	Domain      string `gorm:"size:255;not null;uniqueIndex"`
	Description string `gorm:"type:text"`

	Port     string   `gorm:"size:16;not null"`
	Platform Platform `gorm:"type:text;not null;default:linux/amd64"`

	// Runtime identity
	ContainerName string `gorm:"size:255;not null;uniqueIndex"`
	ContainerID   string `gorm:"size:255"`

	// Image state
	CurrentImageRef string `gorm:"type:text"` // last known good image
	TargetImageRef  string `gorm:"type:text"` // image we want running

	Status        Status        `gorm:"type:text;not null;default:stopped"`
	DesiredStatus DesiredStatus `gorm:"type:text;not null;default:stopped"`
	LastError     string        `gorm:"type:text"`

	IsActive     bool       `gorm:"not null;default:true"`
	LastActiveAt *time.Time `gorm:"index"`

	CreatedAt time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`

	CreatedByID string `gorm:"type:text;not null;index"`
	CreatedBy   User   `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	UpdatedByID string `gorm:"type:text;not null;index"`
	UpdatedBy   User   `gorm:"foreignKey:UpdatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.ContainerName == "" {
		p.ContainerName = fmt.Sprintf("rivet-%s", p.ID)
	}
	return
}

func (s Status) Valid() bool {
	return s == StatusRunning || s == StatusStopped || s == StatusDeploying || s == StatusFailed
}

func (s DesiredStatus) Valid() bool {
	return s == DesiredStatusRunning || s == DesiredStatusStopped
}

func (p Platform) Valid() bool {
	return p == PlatformLinuxAMD64 || p == PlatformLinuxARM64
}
