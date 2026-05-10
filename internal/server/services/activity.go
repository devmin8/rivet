package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/devmin8/rivet/internal/server/database"
	"gorm.io/gorm"
)

const activityFlushInterval = 5 * time.Second
const activityDomainRefreshInterval = 30 * time.Second

type ActivityTracker struct {
	// db is used only when loading domain mappings and flushing last_active_at.
	db           *gorm.DB
	accessLog    string
	serverDomain string
	log          *slog.Logger

	startOnce sync.Once

	// mu protects domains, lastSeen, and lastReload because the tail and flush goroutines both use them.
	mu         sync.Mutex
	domains    map[string]string
	lastSeen   map[string]time.Time
	lastReload time.Time

	lastTailErrorLog time.Time
}

// NewActivityTracker watches Caddy access logs. Those logs identify project
// traffic by HTTP host, so the tracker keeps a small host -> project ID cache.
func NewActivityTracker(db *gorm.DB, accessLog string, serverDomain string, log *slog.Logger) *ActivityTracker {
	return &ActivityTracker{
		db:           db,
		accessLog:    strings.TrimSpace(accessLog),
		serverDomain: NormalizeProjectHost(serverDomain),
		log:          log,
		domains:      make(map[string]string),
		lastSeen:     make(map[string]time.Time),
	}
}

func (t *ActivityTracker) Start(ctx context.Context) {
	t.startOnce.Do(func() {
		if t.accessLog == "" {
			return
		}

		// tail reads Caddy log lines; flushLoop periodically writes the latest activity to SQLite.
		go t.tail(ctx)
		go t.flushLoop(ctx)
	})
}

func (t *ActivityTracker) tail(ctx context.Context) {
	for {
		// tailFile returns when the file cannot be read, is rotated/truncated, or the server shuts down.
		if err := t.tailFile(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.logTailError(err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			// Retry opening the log file after a short delay. This also handles Caddy creating the file late.
		}
	}
}

func (t *ActivityTracker) logTailError(err error) {
	// Avoid writing the same warning every second if Caddy has not created the access log yet.
	if t.log == nil || time.Since(t.lastTailErrorLog) < 30*time.Second {
		return
	}

	t.lastTailErrorLog = time.Now()
	t.log.Warn("failed to tail caddy access log", "path", t.accessLog, "err", err)
}

func (t *ActivityTracker) tailFile(ctx context.Context) error {
	file, err := os.Open(t.accessLog)
	if err != nil {
		return err
	}
	// Ensure the file descriptor is closed when tailFile exits.
	defer file.Close()

	// Keep the original file identity so we can notice Caddy log rotation later.
	initialStat, err := file.Stat()
	if err != nil {
		return err
	}

	// Start at the end: old log lines should not count as fresh project traffic on server startup.
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadBytes('\n')
		if err == nil {
			// One complete JSON log line arrived.
			t.recordAccessLogLine(line)
			continue
		}
		if !errors.Is(err, io.EOF) {
			return err
		}
		// If Caddy rotated the file, access.log now points to a different file. Reopen it.
		if currentStat, statErr := os.Stat(t.accessLog); statErr == nil && !os.SameFile(initialStat, currentStat) {
			return nil
		}
		if offset, seekErr := file.Seek(0, io.SeekCurrent); seekErr == nil {
			// If the active file became smaller than our read offset, it was truncated. Reopen it.
			if currentStat, statErr := os.Stat(t.accessLog); statErr == nil && currentStat.Size() < offset {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			// No new line yet. Sleep briefly, then try reading again.
		}
	}
}

func (t *ActivityTracker) recordAccessLogLine(line []byte) {
	// This struct matches only the Caddy JSON fields we need. Unknown fields are ignored.
	var entry struct {
		Timestamp float64 `json:"ts"`
		Request   struct {
			Host string `json:"host"`
		} `json:"request"`
	}

	// Bad or partial log lines are ignored; activity tracking should never break request handling.
	if err := json.Unmarshal(line, &entry); err != nil {
		return
	}

	host := NormalizeProjectHost(entry.Request.Host)
	if host == "" || host == t.serverDomain {
		return
	}

	projectID, ok := t.projectIDForHost(host)
	if !ok {
		return
	}

	seenAt := time.Now().UTC()
	if entry.Timestamp > 0 {
		// Caddy's ts is seconds with a fractional part. Split it into seconds and nanoseconds for time.Unix.
		secs := int64(entry.Timestamp)
		nanos := int64((entry.Timestamp - float64(secs)) * float64(time.Second))
		seenAt = time.Unix(secs, nanos).UTC()
	}

	// Store only the newest timestamp per project in memory; SQLite is updated by flush().
	t.mu.Lock()
	t.lastSeen[projectID] = seenAt
	t.mu.Unlock()
}

func (t *ActivityTracker) projectIDForHost(host string) (string, bool) {
	t.mu.Lock()
	// Refresh the domain cache occasionally so new/deleted projects are picked up without a restart.
	stale := time.Since(t.lastReload) > activityDomainRefreshInterval
	t.mu.Unlock()

	if stale {
		t.reloadDomains()
	}

	t.mu.Lock()
	projectID, ok := t.domains[host]
	t.mu.Unlock()
	return projectID, ok
}

func (t *ActivityTracker) reloadDomains() {
	var projects []database.Project
	// Load only active projects because inactive/deleted projects should not receive wake/activity behavior.
	if err := t.db.Where("is_active = ?", true).Find(&projects).Error; err != nil {
		if t.log != nil {
			t.log.Warn("failed to load project domains for activity tracking", "err", err)
		}
		return
	}

	// Build a fresh map off to the side, then swap it in under the mutex.
	domains := make(map[string]string, len(projects))
	for _, project := range projects {
		host := NormalizeProjectHost(project.Domain)
		if host != "" {
			domains[host] = project.ID
		}
	}

	t.mu.Lock()
	t.domains = domains
	t.lastReload = time.Now()
	t.mu.Unlock()
}

func (t *ActivityTracker) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(activityFlushInterval)
	// Stop releases ticker resources when this goroutine exits.
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.flush()
			return
		case <-ticker.C:
			t.flush()
		}
	}
}

func (t *ActivityTracker) flush() {
	t.mu.Lock()
	// Swap the map so log tailing can keep recording while DB writes happen outside the lock.
	lastSeen := t.lastSeen
	t.lastSeen = make(map[string]time.Time)
	t.mu.Unlock()

	for projectID, seenAt := range lastSeen {
		if err := t.db.Model(&database.Project{}).
			Where("id = ? AND is_active = ?", projectID, true).
			Update("last_active_at", seenAt).
			Error; err != nil && t.log != nil {
			t.log.Warn("failed to flush project activity", "project_id", projectID, "err", err)
		}
	}
}
