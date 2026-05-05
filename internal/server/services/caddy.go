package services

import (
	"context"
	"time"
)

const caddySyncInterval = 60 * time.Second

func (s *RoutingService) StartSyncer(ctx context.Context) {
	go s.run(ctx)
}

func (s *RoutingService) run(ctx context.Context) {
	ticker := time.NewTicker(caddySyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Sync(ctx); err != nil && s.log != nil {
				s.log.Error("failed to sync caddy routes", "err", err)
			}
		}
	}
}
