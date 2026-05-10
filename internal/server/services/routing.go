package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/devmin8/rivet/internal/caddy"
	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

type caddyLoader interface {
	Load(ctx context.Context, defaultRoute caddy.DefaultRoute, routes []caddy.Route) error
}

type RoutingService struct {
	db           *gorm.DB
	caddy        caddyLoader
	defaultRoute caddy.DefaultRoute
	log          *slog.Logger

	mu       sync.Mutex
	lastHash string
}

func NewRoutingService(db *gorm.DB, caddyClient caddyLoader, serverDomain string, serverPort int, log *slog.Logger) *RoutingService {
	return &RoutingService{
		db:    db,
		caddy: caddyClient,
		defaultRoute: caddy.DefaultRoute{
			Domain:               strings.TrimSpace(serverDomain),
			APIContainerName:     "rivet-server",
			APIPort:              serverPort,
			ConsoleContainerName: "rivet-console",
			ConsolePort:          80,
		},
		log: log,
	}
}

func (s *RoutingService) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	routes, err := s.routes()
	if err != nil {
		return err
	}

	hash := hashRoutes(s.defaultRoute, routes)
	if hash == s.lastHash {
		return nil
	}

	if err := s.caddy.Load(ctx, s.defaultRoute, routes); err != nil {
		return err
	}

	s.lastHash = hash
	if s.log != nil {
		s.log.Info("synced caddy routes", "routes", len(routes))
	}

	return nil
}

func (s *RoutingService) routes() ([]caddy.Route, error) {
	var projects []database.Project

	err := s.db.
		Where("is_active = ? AND status IN ?", true, []database.Status{
			database.StatusRunning,
			database.StatusSleeping,
			database.StatusWaking,
		}).
		Find(&projects).
		Error

	if err != nil {
		return nil, err
	}

	routes := make([]caddy.Route, 0, len(projects))

	for _, project := range projects {
		containerName := project.ContainerName
		portText := project.Port

		if project.Status == database.StatusSleeping || project.Status == database.StatusWaking {
			// Sleeping project traffic must reach rivet-server first so the wake handler
			// can turn the public Host header into a project wake request.
			containerName = s.defaultRoute.APIContainerName
			portText = strconv.Itoa(s.defaultRoute.APIPort)
		}

		port, err := strconv.Atoi(strings.TrimSpace(portText))
		if err != nil {
			return nil, fmt.Errorf("invalid project port: project_id=%s port=%q: %w", project.ID, portText, err)
		}

		routes = append(routes, caddy.Route{
			Domain:        strings.TrimSpace(project.Domain),
			ContainerName: containerName,
			Port:          port,
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Domain == routes[j].Domain {
			return routes[i].ContainerName < routes[j].ContainerName
		}
		return routes[i].Domain < routes[j].Domain
	})

	return routes, nil
}

func hashRoutes(defaultRoute caddy.DefaultRoute, routes []caddy.Route) string {
	h := sha256.New()
	fmt.Fprintf(
		h,
		"%s\x00%s\x00%d\x00%s\x00%d\x00",
		defaultRoute.Domain,
		defaultRoute.APIContainerName,
		defaultRoute.APIPort,
		defaultRoute.ConsoleContainerName,
		defaultRoute.ConsolePort,
	)
	for _, route := range routes {
		fmt.Fprintf(h, "%s\x00%s\x00%d\x00", route.Domain, route.ContainerName, route.Port)
	}
	return hex.EncodeToString(h.Sum(nil))
}
