package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/devmin8/rivet/internal/docker"
	"github.com/devmin8/rivet/internal/server/database"
)

const projectRuntimeStatsTTL = 5 * time.Second

type ProjectRuntimeStatsResponse struct {
	AsOf  time.Time
	Items []ProjectRuntimeStatsItem
}

type ProjectRuntimeStatsItem struct {
	ProjectID              string
	CapturedAt             time.Time
	Stale                  bool
	CPUPercent             float64
	CPUSampleWindowSeconds float64
	MemoryUsageBytes       uint64
	MemoryLimitBytes       uint64
	MemoryPercent          float64
	NetworkRxBytes         uint64
	NetworkTxBytes         uint64
	Pids                   uint64
}

type projectRuntimeStatsCacheEntry struct {
	item      ProjectRuntimeStatsItem
	sample    docker.ContainerStatsSample
	sampledAt time.Time
}

type projectRuntimeStatsResult struct {
	project database.Project
	stats   docker.ContainerStats
	err     error
}

// ProjectRuntimeStats returns recent Docker runtime stats for the user's running projects.
func (s *ProjectService) ProjectRuntimeStats(ctx context.Context, userID string, rawIDs string) (ProjectRuntimeStatsResponse, error) {
	// Start from the database so auth filtering happens before touching Docker.
	projects, err := s.runtimeStatsProjects(userID, rawIDs)
	if err != nil {
		return ProjectRuntimeStatsResponse{}, err
	}

	now := time.Now().UTC()
	// No eligible projects is a valid empty stats response, not an error.
	if len(projects) == 0 {
		return ProjectRuntimeStatsResponse{
			AsOf:  now,
			Items: []ProjectRuntimeStatsItem{},
		}, nil
	}

	// One request refreshes Docker stats at a time. Without this, multiple
	// dashboard requests can all miss the cache and fan out to Docker together.
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	items := make([]ProjectRuntimeStatsItem, 0, len(projects))
	refreshProjects := make([]database.Project, 0, len(projects))
	for _, project := range projects {
		// Serve fresh cache entries immediately and only refresh expired/missing rows.
		cached, ok := s.statsCache[project.ID]
		if ok && now.Sub(cached.sampledAt) <= projectRuntimeStatsTTL {
			items = append(items, cached.item)
			continue
		}
		refreshProjects = append(refreshProjects, project)
	}

	// If everything was cache-fresh, avoid Docker entirely.
	if len(refreshProjects) == 0 {
		return ProjectRuntimeStatsResponse{
			AsOf:  now,
			Items: items,
		}, nil
	}

	// Stats are optional. If Docker is unavailable, return whatever cached data exists.
	if s.docker == nil {
		appendCachedRuntimeStats(&items, s.statsCache, refreshProjects)
		return ProjectRuntimeStatsResponse{
			AsOf:  now,
			Items: items,
		}, nil
	}

	// Check Docker once before fanning out, so a down daemon does not create many failures.
	if err := s.docker.CheckRunning(ctx); err != nil {
		appendCachedRuntimeStats(&items, s.statsCache, refreshProjects)
		return ProjectRuntimeStatsResponse{
			AsOf:  now,
			Items: items,
		}, nil
	}

	previousSamples := make(map[string]*docker.ContainerStatsSample, len(refreshProjects))
	// Build the previous-sample map used to calculate CPU without blocking Docker.
	for _, project := range refreshProjects {
		if cached, ok := s.statsCache[project.ID]; ok && !cached.sample.Read.IsZero() {
			// CPU percent needs two cumulative Docker counter samples. Keep the old
			// sample even after the display row expires so the next live refresh can
			// calculate CPU without asking Docker to wait for a second sample.
			sample := cached.sample
			previousSamples[project.ID] = &sample
		}
	}

	// Merge live results into the response and update the cache for the next poll.
	for _, result := range s.collectRuntimeStats(ctx, refreshProjects, previousSamples) {
		if result.err != nil {
			appendCachedRuntimeStat(&items, s.statsCache, result.project.ID)
			s.markRuntimeStatsFailure(result.project, result.err)
			s.logRuntimeStatsFailure(result.project, result.err)
			continue
		}

		item := ProjectRuntimeStatsItem{
			ProjectID:              result.project.ID,
			CapturedAt:             now,
			CPUPercent:             result.stats.CPUPercent,
			CPUSampleWindowSeconds: result.stats.CPUSampleWindowSeconds,
			MemoryUsageBytes:       result.stats.MemoryUsageBytes,
			MemoryLimitBytes:       result.stats.MemoryLimitBytes,
			MemoryPercent:          result.stats.MemoryPercent,
			NetworkRxBytes:         result.stats.NetworkRxBytes,
			NetworkTxBytes:         result.stats.NetworkTxBytes,
			Pids:                   result.stats.Pids,
		}
		s.statsCache[result.project.ID] = projectRuntimeStatsCacheEntry{
			item:      item,
			sample:    result.stats.Sample,
			sampledAt: now,
		}
		items = append(items, item)
	}

	return ProjectRuntimeStatsResponse{
		AsOf:  now,
		Items: items,
	}, nil
}

// collectRuntimeStats reads Docker stats for all selected projects in parallel.
func (s *ProjectService) collectRuntimeStats(ctx context.Context, projects []database.Project, previousSamples map[string]*docker.ContainerStatsSample) []projectRuntimeStatsResult {
	results := make([]projectRuntimeStatsResult, len(projects))
	var wg sync.WaitGroup
	for i, project := range projects {
		wg.Add(1)
		go func(i int, project database.Project) {
			defer wg.Done()
			// Stats reads are independent per container, so collect them in parallel.
			// A 50-project dashboard should wait for the slowest container read once,
			// not pay one Docker round trip after another.
			stats, err := s.docker.ContainerStats(ctx, project.ContainerName, previousSamples[project.ID])
			results[i] = projectRuntimeStatsResult{
				project: project,
				stats:   stats,
				err:     err,
			}
		}(i, project)
	}
	wg.Wait()

	return results
}

// runtimeStatsProjects returns the authenticated user's active projects that may have live stats.
func (s *ProjectService) runtimeStatsProjects(userID string, rawIDs string) ([]database.Project, error) {
	var projects []database.Project
	query := s.db.Where(
		"created_by_id = ? AND is_active = ? AND status = ?",
		userID,
		true,
		database.StatusRunning,
	)

	ids := projectRuntimeStatsIDs(rawIDs)
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}

	if err := query.Find(&projects).Error; err != nil {
		return nil, err
	}

	return projects, nil
}

// projectRuntimeStatsIDs parses the optional comma-separated project ID filter.
func projectRuntimeStatsIDs(rawIDs string) []string {
	parts := strings.Split(rawIDs, ",")
	ids := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids
}

// appendCachedRuntimeStats appends any cached rows for projects that could not be refreshed.
func appendCachedRuntimeStats(items *[]ProjectRuntimeStatsItem, cache map[string]projectRuntimeStatsCacheEntry, projects []database.Project) {
	for _, project := range projects {
		appendCachedRuntimeStat(items, cache, project.ID)
	}
}

// appendCachedRuntimeStat appends one cached row if it exists.
func appendCachedRuntimeStat(items *[]ProjectRuntimeStatsItem, cache map[string]projectRuntimeStatsCacheEntry, projectID string) {
	cached, ok := cache[projectID]
	if !ok {
		return
	}

	item := cached.item
	item.Stale = true
	*items = append(*items, item)
}

func (s *ProjectService) clearRuntimeStatsCache(projectID string) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	delete(s.statsCache, projectID)
}

func (s *ProjectService) markRuntimeStatsFailure(project database.Project, err error) {
	if project.Status != database.StatusRunning {
		return
	}
	if !errors.Is(err, docker.ErrContainerNotFound) && !errors.Is(err, docker.ErrContainerNotRunning) {
		return
	}

	_ = s.markRuntimeFailed(project.ID, project.ContainerID, err)
}

// logRuntimeStatsFailure logs expected missing-container states quietly and real Docker failures as warnings.
func (s *ProjectService) logRuntimeStatsFailure(project database.Project, err error) {
	if s.log == nil {
		return
	}
	if errors.Is(err, docker.ErrContainerNotFound) || errors.Is(err, docker.ErrContainerNotRunning) {
		s.log.Debug("project runtime stats unavailable", "project_id", project.ID, "container_name", project.ContainerName, "err", err)
		return
	}

	s.log.Warn("failed to collect project runtime stats", "project_id", project.ID, "container_name", project.ContainerName, "err", err)
}
