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

type AppStatus string

const (
	AppStatusCreating AppStatus = "creating"
	AppStatusRunning  AppStatus = "running"
	AppStatusStopped  AppStatus = "stopped"
	AppStatusFailed   AppStatus = "failed"
)

type DesiredStatus string

const (
	DesiredStatusRunning DesiredStatus = "running"
	DesiredStatusStopped DesiredStatus = "stopped"
)

type DeploymentStatus string

const (
	DeploymentStatusPending DeploymentStatus = "pending"
	DeploymentStatusRunning DeploymentStatus = "running"
	DeploymentStatusFailed  DeploymentStatus = "failed"
	DeploymentStatusStopped DeploymentStatus = "stopped"
)

type Platform string

const (
	PlatformLinuxAMD64 Platform = "linux/amd64"
	PlatformLinuxARM64 Platform = "linux/arm64"
)

type Project struct {
	ID          string    `gorm:"primaryKey;type:text"`
	Name        string    `gorm:"size:255;not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"not null;autoUpdateTime"`
	CreatedByID string    `gorm:"type:text;not null;index"`
	CreatedBy   User      `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	UpdatedByID string    `gorm:"type:text;not null;index"`
	UpdatedBy   User      `gorm:"foreignKey:UpdatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return
}

type App struct {
	ID                  string        `gorm:"primaryKey;type:text"`
	ProjectID           string        `gorm:"type:text;not null;index"`
	Project             Project       `gorm:"foreignKey:ProjectID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Name                string        `gorm:"size:255;not null"`
	Domain              string        `gorm:"size:255;not null;uniqueIndex"`
	Description         string        `gorm:"type:text"`
	Port                string        `gorm:"size:16;not null"`
	Platform            Platform      `gorm:"type:text;not null;default:linux/amd64"`
	Status              AppStatus     `gorm:"type:text;not null;default:stopped"`
	DesiredStatus       DesiredStatus `gorm:"type:text;not null;default:stopped"`
	StatusUpdatedAt     time.Time     `gorm:"not null;index"`
	CurrentDeploymentID *string       `gorm:"type:text;index"`
	LastActiveAt        *time.Time    `gorm:"index"`
	CreatedAt           time.Time     `gorm:"not null;autoCreateTime"`
	UpdatedAt           time.Time     `gorm:"not null;autoUpdateTime"`
	CreatedByID         string        `gorm:"type:text;not null;index"`
	CreatedBy           User          `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	UpdatedByID         string        `gorm:"type:text;not null;index"`
	UpdatedBy           User          `gorm:"foreignKey:UpdatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (a *App) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.StatusUpdatedAt.IsZero() {
		a.StatusUpdatedAt = time.Now().UTC()
	}
	return
}

type Deployment struct {
	ID              string           `gorm:"primaryKey;type:text"`
	AppID           string           `gorm:"type:text;not null;index"`
	App             App              `gorm:"foreignKey:AppID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Image           string           `gorm:"type:text;not null"`
	ContainerID     string           `gorm:"size:255"`
	Status          DeploymentStatus `gorm:"type:text;not null;default:pending"`
	StatusUpdatedAt time.Time        `gorm:"not null;index"`
	Error           string           `gorm:"type:text"`
	CreatedAt       time.Time        `gorm:"not null;autoCreateTime"`
	UpdatedAt       time.Time        `gorm:"not null;autoUpdateTime"`
	CreatedByID     string           `gorm:"type:text;not null;index"`
	CreatedBy       User             `gorm:"foreignKey:CreatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	UpdatedByID     string           `gorm:"type:text;not null;index"`
	UpdatedBy       User             `gorm:"foreignKey:UpdatedByID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (d *Deployment) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	if d.StatusUpdatedAt.IsZero() {
		d.StatusUpdatedAt = time.Now().UTC()
	}
	return
}

func (s AppStatus) Valid() bool {
	return s == AppStatusCreating || s == AppStatusRunning || s == AppStatusStopped || s == AppStatusFailed
}

func (s DesiredStatus) Valid() bool {
	return s == DesiredStatusRunning || s == DesiredStatusStopped
}

func (s DeploymentStatus) Valid() bool {
	return s == DeploymentStatusPending || s == DeploymentStatusRunning || s == DeploymentStatusFailed || s == DeploymentStatusStopped
}

func (p Platform) Valid() bool {
	return p == PlatformLinuxAMD64 || p == PlatformLinuxARM64
}
