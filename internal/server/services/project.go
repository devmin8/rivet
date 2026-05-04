package services

import (
	"errors"
	"strings"

	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

var ErrProjectNotFound = errors.New("project not found")

type ProjectService struct {
	db *gorm.DB
}

type CreateProjectRequest struct {
	Name        string
	Description string
	CreatedByID string
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) CreateProject(req CreateProjectRequest) (*database.Project, error) {
	project := &database.Project{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		CreatedByID: req.CreatedByID,
		UpdatedByID: req.CreatedByID,
	}

	if err := s.db.Create(project).Error; err != nil {
		return nil, err
	}

	return project, nil
}

func (s *ProjectService) GetProject(id string, userID string) (*database.Project, error) {
	var project database.Project
	if err := s.db.Where("id = ? AND created_by_id = ?", id, userID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	return &project, nil
}

func (s *ProjectService) ListProjects(userID string) ([]database.Project, error) {
	var projects []database.Project
	if err := s.db.Where("created_by_id = ?", userID).Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}

	return projects, nil
}

func (s *ProjectService) DeleteProject(id string, userID string) error {
	result := s.db.Where("id = ? AND created_by_id = ?", id, userID).Delete(&database.Project{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}

	return nil
}
