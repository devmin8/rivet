package services

import (
	"strconv"
	"strings"

	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

type ProjectService struct {
	db *gorm.DB
}

type CreateProjectRequest struct {
	Name        string
	Domain      string
	Description string
	Port        uint32
	Image       string
	CreatedByID string
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) CreateProject(req CreateProjectRequest) (*database.Project, error) {
	project := &database.Project{
		Name:        strings.TrimSpace(req.Name),
		Domain:      strings.TrimSpace(req.Domain),
		Description: strings.TrimSpace(req.Description),
		Port:        strconv.FormatUint(uint64(req.Port), 10),
		Image:       strings.TrimSpace(req.Image),
		Status:      database.StatusCreating,
		CreatedByID: req.CreatedByID,
		UpdatedByID: req.CreatedByID,
	}

	if err := s.db.Create(project).Error; err != nil {
		return nil, err
	}

	return project, nil
}
