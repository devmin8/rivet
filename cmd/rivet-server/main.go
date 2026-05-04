package main

import (
	"log"
	"os"

	"github.com/devmin8/rivet/internal/logger"
	"github.com/devmin8/rivet/internal/server"
	"github.com/devmin8/rivet/internal/server/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	log := logger.New()

	server, err := server.New(cfg, log)
	if err != nil {
		log.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := server.Run(); err != nil {
		log.Error("failed to run server", "error", err)
		os.Exit(1)
	}
}
