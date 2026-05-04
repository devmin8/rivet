package services

import (
	"errors"
	"strconv"
	"strings"

	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

var ErrProjectNotFound = errors.New("project not found")
var ErrProjectInactive = errors.New("project is not active")

type ProjectService struct {
	db *gorm.DB
}

type CreateProjectRequest struct {
	Name        string
	Domain      string
	Description string
	Port        uint32
	Image       string
	Platform    string
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
		Platform:    normalizePlatform(req.Platform),
		Status:      database.StatusCreating,
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

	if !project.IsActive {
		return nil, ErrProjectInactive
	}

	return &project, nil
}

func (s *ProjectService) UpdateProjectImage(id string, userID string, image string) (*database.Project, error) {
	project, err := s.GetProject(id, userID)
	if err != nil {
		return nil, err
	}

	project.Image = strings.TrimSpace(image)
	project.Status = database.StatusStopped
	project.UpdatedByID = userID

	if err := s.db.Save(project).Error; err != nil {
		return nil, err
	}

	return project, nil
}

func normalizePlatform(platform string) database.Platform {
	value := database.Platform(strings.TrimSpace(platform))
	if value.Valid() {
		return value
	}

	return database.PlatformLinuxAMD64
}
