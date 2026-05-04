package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/cli/client"
)

const (
	defaultServerURL = "http://localhost:3000"

	serverURLEnv = "RIVET_SERVER_URL"
	timeoutEnv   = "RIVET_API_TIMEOUT"
)

type cliConfig struct {
	ServerURL string
	Timeout   time.Duration
}

func loadConfig() (cliConfig, error) {
	cfg := cliConfig{
		ServerURL: strings.TrimSpace(os.Getenv(serverURLEnv)),
		Timeout:   client.DefaultTimeout,
	}

	if cfg.ServerURL == "" {
		cfg.ServerURL = defaultServerURL
	}

	rawTimeout := strings.TrimSpace(os.Getenv(timeoutEnv))
	if rawTimeout == "" {
		return cfg, nil
	}

	timeout, err := time.ParseDuration(rawTimeout)
	if err != nil {
		return cliConfig{}, fmt.Errorf("%s must be a valid duration: %w", timeoutEnv, err)
	}
	if timeout <= 0 {
		return cliConfig{}, fmt.Errorf("%s must be greater than zero", timeoutEnv)
	}

	cfg.Timeout = timeout
	return cfg, nil
}
