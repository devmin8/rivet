package server

import (
	"github.com/devmin8/rivet/internal/logger"
	"github.com/devmin8/rivet/internal/server/config"
	"github.com/devmin8/rivet/internal/server/web"
)

func Run() error {
	logger := logger.NewLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return err
	}

	return web.NewServer(cfg, logger)
}
