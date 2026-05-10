package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/devmin8/rivet/internal/server/config"
)

type Route struct {
	Domain        string
	ContainerName string
	Port          int
}

// AccessLogPath is inside the Caddy container. The install scripts mount the same host
// directory into rivet-server at CADDY_ACCESS_LOG_PATH so the server can tail it.
// Caddy file logs roll by default, so this file should not grow forever.
const AccessLogPath = "/var/log/caddy/access.log"

type DefaultRoute struct {
	Domain               string
	APIContainerName     string
	APIPort              int
	ConsoleContainerName string
	ConsolePort          int
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
	AppEnv  config.AppEnv
}

func New(baseURL string, appEnv config.AppEnv) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
		},
		AppEnv: appEnv,
	}
}

func (c *Client) Load(ctx context.Context, defaultRoute DefaultRoute, routes []Route) error {
	cfg := buildCaddyConfig(c.AppEnv, defaultRoute, routes)

	body, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/load", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("caddy load failed: status=%d body=%s", res.StatusCode, string(b))
	}

	return nil
}

func buildCaddyConfig(env config.AppEnv, defaultRoute DefaultRoute, routes []Route) map[string]any {
	sorted := append([]Route(nil), routes...)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Domain < sorted[j].Domain
	})

	caddyRoutes := make([]any, 0, len(sorted)+2)
	caddyRoutes = append(caddyRoutes, defaultAppRoute(defaultRoute))

	for _, r := range sorted {
		caddyRoutes = append(caddyRoutes, projectRoute(r))
	}
	caddyRoutes = append(caddyRoutes, notFoundRoute())

	server := map[string]any{
		"listen": []string{":80"},
		"routes": caddyRoutes,
		"logs": map[string]any{
			"default_logger_name": "rivet_access",
		},
	}

	if env == config.Dev {
		server["automatic_https"] = map[string]any{"disable": true}
	} else {
		server["listen"] = []string{":80", ":443"}
	}

	return map[string]any{
		"logging": map[string]any{
			"logs": map[string]any{
				"default": map[string]any{
					"exclude": []string{"http.log.access.rivet_access"},
				},
				"rivet_access": map[string]any{
					"writer": map[string]any{
						"output":   "file",
						"filename": AccessLogPath,
					},
					"encoder": map[string]any{
						"format": "json",
					},
					"include": []string{"http.log.access.rivet_access"},
				},
			},
		},
		"apps": map[string]any{
			"http": map[string]any{
				"servers": map[string]any{
					"rivet": server,
				},
			},
		},
	}
}

func defaultAppRoute(r DefaultRoute) map[string]any {
	return map[string]any{
		"match": []any{
			map[string]any{
				"host": []string{r.Domain},
			},
		},
		"handle": []any{
			map[string]any{
				"handler": "subroute",
				"routes": []any{
					map[string]any{
						"match": []any{
							map[string]any{
								"path": []string{"/api", "/api/*"},
							},
						},
						"handle":   []any{reverseProxy(r.APIContainerName, r.APIPort)},
						"terminal": true,
					},
					map[string]any{
						"handle":   []any{reverseProxy(r.ConsoleContainerName, r.ConsolePort)},
						"terminal": true,
					},
				},
			},
		},
		"terminal": true,
	}
}

func projectRoute(r Route) map[string]any {
	return map[string]any{
		"match": []any{
			map[string]any{
				"host": []string{r.Domain},
			},
		},
		"handle":   []any{reverseProxy(r.ContainerName, r.Port)},
		"terminal": true,
	}
}

func reverseProxy(containerName string, port int) map[string]any {
	return map[string]any{
		"handler": "reverse_proxy",
		"upstreams": []any{
			map[string]any{
				"dial": fmt.Sprintf("%s:%d", containerName, port),
			},
		},
	}
}
