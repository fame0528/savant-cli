package tools

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// ForgeCurator manages the lifecycle of forged tools.
type ForgeCurator struct {
	forgeDir       string
	provenance     *ProvenanceTracker
	staleThreshold time.Duration
}

// NewForgeCurator creates a curator for the forge directory.
func NewForgeCurator(forgeDir string, provenance *ProvenanceTracker) *ForgeCurator {
	return &ForgeCurator{
		forgeDir:       forgeDir,
		provenance:     provenance,
		staleThreshold: 30 * 24 * time.Hour, // 30 days
	}
}

// RunAutoTransitions scans for stale tools and archives them.
func (fc *ForgeCurator) RunAutoTransitions() int {
	slog.Info("Curator: scanning for stale forged tools")
	count := 0

	entries, err := os.ReadDir(fc.forgeDir)
	if err != nil {
		return 0
	}

	archiveDir := filepath.Join(fc.forgeDir, ".archive")

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".archive" {
			continue
		}

		name := entry.Name()
		lastActivity := fc.provenance.LastActivity(name)
		if lastActivity == nil {
			continue
		}

		// Check if pinned
		entries := fc.provenance.Replay()
		pinned := false
		for _, e := range entries {
			if e.Name == name && e.Action == "pin" && e.Pinned != nil && *e.Pinned {
				pinned = true
				break
			}
		}
		if pinned {
			continue
		}

		// Check staleness
		elapsed := time.Since(*lastActivity)
		if elapsed > fc.staleThreshold {
			// Archive
			os.MkdirAll(archiveDir, 0o755)
			src := filepath.Join(fc.forgeDir, name)
			dst := filepath.Join(archiveDir, name)
			if err := os.Rename(src, dst); err != nil {
				slog.Warn("Curator: failed to archive", "tool", name, "error", err)
				continue
			}

			// Record in provenance
			fc.provenance.Append(ProvenanceEntry{
				Name:      name,
				Action:    "archive",
				Reason:    fmt.Sprintf("Stale for %d days", int(elapsed.Hours()/24)),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})

			slog.Info("Curator: archived stale tool", "tool", name, "days", int(elapsed.Hours()/24))
			count++
		}
	}

	if count > 0 {
		slog.Info("Curator: archived tools", "count", count)
	}
	return count
}
