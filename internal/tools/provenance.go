package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ProvenanceEntry represents a single lifecycle event for a forged tool.
type ProvenanceEntry struct {
	Name           string  `json:"name"`
	CreatorAgentID string  `json:"creator_agent_id"`
	Action         string  `json:"action"` // forge, patch, archive, pin, rollback, rate
	Version        string  `json:"version,omitempty"`
	Description    string  `json:"description,omitempty"`
	Category       string  `json:"category,omitempty"`
	Rating         string  `json:"rating,omitempty"` // thumbs_up, thumbs_down
	Comment        string  `json:"comment,omitempty"`
	Pinned         *bool   `json:"pinned,omitempty"`
	Reason         string  `json:"reason,omitempty"`
	SupersededBy   string  `json:"superseded_by,omitempty"`
	AuditResult    string  `json:"audit_result,omitempty"`
	FromVersion    string  `json:"from_version,omitempty"`
	ToVersion      string  `json:"to_version,omitempty"`
	Timestamp      string  `json:"timestamp"`
}

// ToolStats computes usage statistics for a forged tool.
type ToolStats struct {
	UseCount     int        `json:"use_count"`
	UniqueAgents int        `json:"unique_agents"`
	ThumbsUp     int        `json:"thumbs_up"`
	ThumbsDown   int        `json:"thumbs_down"`
	SuccessRate  float64    `json:"success_rate"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
}

// ProvenanceTracker manages the JSONL provenance log.
type ProvenanceTracker struct {
	mu   sync.Mutex
	path string
}

// NewProvenanceTracker creates a tracker at the given path.
func NewProvenanceTracker(logPath string) *ProvenanceTracker {
	dir := filepath.Dir(logPath)
	os.MkdirAll(dir, 0o755)
	return &ProvenanceTracker{path: logPath}
}

// Append adds an entry to the provenance log.
func (pt *ProvenanceTracker) Append(entry ProvenanceEntry) error {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	pt.mu.Lock()
	defer pt.mu.Unlock()

	f, err := os.OpenFile(pt.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(line)
	return err
}

// Replay reads all entries from the provenance log.
func (pt *ProvenanceTracker) Replay() []ProvenanceEntry {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	data, err := os.ReadFile(pt.path)
	if err != nil {
		return nil
	}

	var entries []ProvenanceEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry ProvenanceEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

// ComputeStats computes usage statistics for a named tool.
func (pt *ProvenanceTracker) ComputeStats(name string) ToolStats {
	entries := pt.Replay()
	agents := make(map[string]bool)
	var lastUsed *time.Time

	stats := ToolStats{SuccessRate: 1.0}

	for _, e := range entries {
		if e.Name != name {
			continue
		}
		switch e.Action {
		case "rate":
			stats.UseCount++
			if e.CreatorAgentID != "" {
				agents[e.CreatorAgentID] = true
			}
			if e.Rating == "thumbs_up" {
				stats.ThumbsUp++
			} else if e.Rating == "thumbs_down" {
				stats.ThumbsDown++
			}
			t, err := time.Parse(time.RFC3339, e.Timestamp)
			if err == nil {
				lastUsed = &t
			}
		case "forge", "patch":
			t, err := time.Parse(time.RFC3339, e.Timestamp)
			if err == nil {
				lastUsed = &t
			}
		}
	}

	stats.UniqueAgents = len(agents)
	stats.LastUsedAt = lastUsed

	total := stats.ThumbsUp + stats.ThumbsDown
	if total > 0 {
		stats.SuccessRate = float64(stats.ThumbsUp) / float64(total)
	}
	return stats
}

// LastActivity returns the timestamp of the last activity for a tool.
func (pt *ProvenanceTracker) LastActivity(name string) *time.Time {
	entries := pt.Replay()
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Name == name {
			t, err := time.Parse(time.RFC3339, entries[i].Timestamp)
			if err == nil {
				return &t
			}
		}
	}
	return nil
}
