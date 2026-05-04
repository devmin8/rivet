package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/config"
	"github.com/devmin8/rivet/internal/server/database"
	"github.com/devmin8/rivet/internal/server/services"
	"github.com/devmin8/rivet/internal/server/web"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type App struct {
	cfg    *config.ServerEnv
	db     *gorm.DB
	docker *docker.Client
	log    *slog.Logger
}

func New(cfg *config.ServerEnv, log *slog.Logger) (*App, error) {
	db, err := database.New(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	// For initial development phases, we'll auto migrate the database
	if err := database.AutoMigrate(db); err != nil {
		return nil, err
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, errors.Join(err, closeDB(db))
	}

	return &App{cfg: cfg, db: db, docker: dockerClient, log: log}, nil
}

func (s *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := web.NewServer(s.cfg, s.db, s.docker, s.log)
	app.Hooks().OnListen(func(data fiber.ListenData) error {
		s.log.Info("🚀 web server started", "host", data.Host, "port", data.Port, "pid", data.PID)
		return nil
	})

	projectService := services.NewProjectService(s.db, s.docker)
	go projectService.RunProjectReconciler(ctx, time.Minute, 10*time.Minute)

	// start web server in a separate goroutine
	errCh := make(chan error, 1)
	go func() {
		addr := ":" + s.cfg.Port
		s.log.Info("📡 web server starting", "addr", addr)

		if err := app.Listen(addr); err != nil && !isExpectedClose(err) {
			errCh <- err
			return
		}

		// normal shutdown
		errCh <- nil
	}()

	// wait for shutdown signal
	var runErr error
	select {
	case <-ctx.Done():
		s.log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			s.log.Error("server error", "err", err)
			runErr = err
		}
	}

	return errors.Join(runErr, s.shutdown(app))
}

func (s *App) close() error {
	var closeErr error
	if s.docker != nil {
		closeErr = errors.Join(closeErr, s.docker.Close())
	}

	return errors.Join(closeErr, closeDB(s.db))
}

func (s *App) shutdown(app interface {
	ShutdownWithContext(context.Context) error
}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.log.Info("shutting down server")

	// shutdown web server
	var shutdownErr error
	if err := app.ShutdownWithContext(ctx); err != nil {
		s.log.Error("failed to shutdown web server", "err", err)
		shutdownErr = errors.Join(shutdownErr, err)
	}

	// close database connection
	if err := s.close(); err != nil {
		s.log.Error("failed to close database", "err", err)
		shutdownErr = errors.Join(shutdownErr, err)
	}

	s.log.Info("shutdown complete")
	return shutdownErr
}

// helper function to check if the error is expected
func isExpectedClose(err error) bool {
	return errors.Is(err, net.ErrClosed) ||
		errors.Is(err, context.Canceled)
}

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
