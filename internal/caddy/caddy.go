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

func (c *Client) Load(ctx context.Context, routes []Route) error {
	cfg := buildCaddyConfig(c.AppEnv, routes)

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

func buildCaddyConfig(env config.AppEnv, routes []Route) map[string]any {
	sorted := append([]Route(nil), routes...)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Domain < sorted[j].Domain
	})

	caddyRoutes := make([]any, 0, len(sorted))

	for _, r := range sorted {
		caddyRoutes = append(caddyRoutes, map[string]any{
			"match": []any{
				map[string]any{
					"host": []string{r.Domain},
				},
			},
			"handle": []any{
				map[string]any{
					"handler": "reverse_proxy",
					"upstreams": []any{
						map[string]any{
							"dial": fmt.Sprintf("%s:%d", r.ContainerName, r.Port),
						},
					},
				},
			},
			"terminal": true,
		})
	}
	caddyRoutes = append(caddyRoutes, notFoundRoute())

	server := map[string]any{
		"listen": []string{":80"},
		"routes": caddyRoutes,
	}

	if env == config.Dev {
		server["automatic_https"] = map[string]any{"disable": true}
	} else {
		server["listen"] = []string{":80", ":443"}
	}

	return map[string]any{
		"apps": map[string]any{
			"http": map[string]any{
				"servers": map[string]any{
					"rivet": server,
				},
			},
		},
	}
}
