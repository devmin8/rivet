package database

import (
	"path/filepath"

	"github.com/devmin8/rivet/internal/fsutil"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func New(dbPath string) (*gorm.DB, error) {
	if err := fsutil.EnsureDir(filepath.Dir(dbPath)); err != nil {
		return nil, err
	}

	return gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Session{},
		&Project{},
		&App{},
		&Deployment{},
	)
}
