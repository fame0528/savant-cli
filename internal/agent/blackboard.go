// Package agent - Blackboard provides a thread-safe shared state store.
// The blackboard solves the "island problem": sub-agents spawned by the parent
// need context about what files were read/modified, what decisions were made,
// and what the overall plan is. Instead of passing raw conversation history
// (expensive), the blackboard holds structured shared state.
package agent

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// BlackboardKey is a typed string for blackboard keys.
type BlackboardKey string

const (
	// BlackboardPlan is the current approach/strategy. Written by parent.
	BlackboardPlan BlackboardKey = "plan"
	// BlackboardFilesModified is the list of files changed this session.
	BlackboardFilesModified BlackboardKey = "files_modified"
	// BlackboardFilesRead is the list of files read this session.
	BlackboardFilesRead BlackboardKey = "files_read"
	// BlackboardDecisions is the list of key decisions made.
	BlackboardDecisions BlackboardKey = "decisions"
	// BlackboardBlockers is the list of open issues.
	BlackboardBlockers BlackboardKey = "blockers"
	// BlackboardCwd is the working directory.
	BlackboardCwd BlackboardKey = "cwd"
	// BlackboardGoal is the top-level user request.
	BlackboardGoal BlackboardKey = "goal"
	// BlackboardAgentType is the current agent's type (code/explore/review).
	BlackboardAgentType BlackboardKey = "agent_type"
)

// validKeys contains all valid blackboard keys for validation.
var validKeys = map[BlackboardKey]bool{
	BlackboardPlan:          true,
	BlackboardFilesModified: true,
	BlackboardFilesRead:    true,
	BlackboardDecisions:    true,
	BlackboardBlockers:     true,
	BlackboardCwd:          true,
	BlackboardGoal:         true,
	BlackboardAgentType:    true,
}

// BlackboardEntry holds a single blackboard value with metadata.
type BlackboardEntry struct {
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"` // Agent identifier
}

// BlackboardSnapshot is a point-in-time copy of all blackboard state.
type BlackboardSnapshot struct {
	Entries map[BlackboardKey][]BlackboardEntry `json:"entries"`
	At      time.Time                           `json:"at"`
}

// Blackboard is a thread-safe shared state store.
// All agents read/write the blackboard to share context without passing
// raw conversation history.
type Blackboard struct {
	mu      sync.RWMutex
	entries map[BlackboardKey][]BlackboardEntry
}

// NewBlackboard creates an empty blackboard.
func NewBlackboard() *Blackboard {
	return &Blackboard{
		entries: make(map[BlackboardKey][]BlackboardEntry),
	}
}

// Set sets a singleton key (plan, cwd, goal, agent_type).
// These keys hold a single value; calling Set replaces the value.
func (b *Blackboard) Set(key BlackboardKey, value, updatedBy string) error {
	if !validKeys[key] {
		return fmt.Errorf("invalid blackboard key: %s", key)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Singleton keys replace all existing entries
	b.entries[key] = []BlackboardEntry{{
		Value:     value,
		UpdatedAt: time.Now(),
		UpdatedBy: updatedBy,
	}}
	return nil
}

// Get returns the most recent value for a singleton key.
// Returns empty string if the key has no entries.
func (b *Blackboard) Get(key BlackboardKey) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entries, ok := b.entries[key]
	if !ok || len(entries) == 0 {
		return ""
	}
	return entries[len(entries)-1].Value
}

// GetAll returns all entries for a key.
func (b *Blackboard) GetAll(key BlackboardKey) []BlackboardEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entries, ok := b.entries[key]
	if !ok {
		return nil
	}
	result := make([]BlackboardEntry, len(entries))
	copy(result, entries)
	return result
}

// Append appends a value to a list key (files_modified, files_read, decisions, blockers).
// These keys hold ordered lists; calling Append adds to the list.
// Duplicate detection: for files_modified and files_read, duplicates are ignored
// (same value, same key). For decisions and blockers, all entries are kept.
func (b *Blackboard) Append(key BlackboardKey, value, updatedBy string) error {
	if !validKeys[key] {
		return fmt.Errorf("invalid blackboard key: %s", key)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// For file lists, skip duplicates
	if key == BlackboardFilesModified || key == BlackboardFilesRead {
		for i, entry := range b.entries[key] {
			if entry.Value == value {
				// Update the timestamp but don't duplicate
				b.entries[key][i].UpdatedAt = time.Now()
				return nil
			}
		}
	}

	b.entries[key] = append(b.entries[key], BlackboardEntry{
		Value:     value,
		UpdatedAt: time.Now(),
		UpdatedBy: updatedBy,
	})

	// Enforce max list length to prevent unbounded growth
	maxEntries := 100
	if len(b.entries[key]) > maxEntries {
		// Keep only the most recent entries
		b.entries[key] = b.entries[key][len(b.entries[key])-maxEntries:]
	}

	return nil
}

// List returns all values for a list key as a slice of strings.
func (b *Blackboard) List(key BlackboardKey) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entries, ok := b.entries[key]
	if !ok {
		return nil
	}
	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.Value
	}
	return result
}

// Snapshot returns a point-in-time copy of all blackboard state.
func (b *Blackboard) Snapshot() BlackboardSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entries := make(map[BlackboardKey][]BlackboardEntry)
	for k, v := range b.entries {
		entries[k] = append([]BlackboardEntry(nil), v...)
	}

	return BlackboardSnapshot{
		Entries: entries,
		At:      time.Now(),
	}
}

// FormatSnapshot formats the blackboard snapshot as a readable string
// suitable for inclusion in a sub-agent's system prompt.
func (s BlackboardSnapshot) FormatSnapshot() string {
	if len(s.Entries) == 0 {
		return "(empty)"
	}

	var sb strings.Builder

	// Display order: goal, plan, files modified, files read, decisions, blockers
	type keyDisplay struct {
		key   BlackboardKey
		label string
	}

	order := []keyDisplay{
		{BlackboardGoal, "Goal"},
		{BlackboardPlan, "Plan"},
		{BlackboardCwd, "Working Directory"},
		{BlackboardFilesModified, "Files Modified"},
		{BlackboardFilesRead, "Files Read"},
		{BlackboardDecisions, "Decisions"},
		{BlackboardBlockers, "Blockers"},
	}

	for _, kd := range order {
		entries, ok := s.Entries[kd.key]
		if !ok || len(entries) == 0 {
			continue
		}

		switch kd.key {
		case BlackboardGoal, BlackboardPlan, BlackboardCwd:
			// Singleton values
			latest := entries[len(entries)-1]
			sb.WriteString(fmt.Sprintf("%s: %s\n", kd.label, latest.Value))
		case BlackboardFilesModified, BlackboardFilesRead:
			// File lists - show sorted unique files
			seen := make(map[string]bool)
			var files []string
			for _, e := range entries {
				if !seen[e.Value] {
					seen[e.Value] = true
					files = append(files, e.Value)
				}
			}
			sort.Strings(files)
			if len(files) > 20 {
				files = files[:20]
				sb.WriteString(fmt.Sprintf("%s (%d total, showing 20):\n", kd.label, len(seen)))
			} else {
				sb.WriteString(fmt.Sprintf("%s:\n", kd.label))
			}
			for _, f := range files {
				sb.WriteString(fmt.Sprintf("  - %s\n", f))
			}
		case BlackboardDecisions:
			// Decision list - show most recent first, max 10
			sb.WriteString(fmt.Sprintf("%s:\n", kd.label))
			start := 0
			if len(entries) > 10 {
				start = len(entries) - 10
				sb.WriteString(fmt.Sprintf("  (showing last %d of %d)\n", 10, len(entries)))
			}
			for i := start; i < len(entries); i++ {
				sb.WriteString(fmt.Sprintf("  - %s\n", entries[i].Value))
			}
		case BlackboardBlockers:
			// Blocker list - show all
			sb.WriteString(fmt.Sprintf("%s:\n", kd.label))
			for _, e := range entries {
				sb.WriteString(fmt.Sprintf("  - %s\n", e.Value))
			}
		}
	}

	return sb.String()
}

// Clear removes all entries from the blackboard.
func (b *Blackboard) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = make(map[BlackboardKey][]BlackboardEntry)
}

// Delete removes all entries for a specific key.
func (b *Blackboard) Delete(key BlackboardKey) error {
	if !validKeys[key] {
		return fmt.Errorf("invalid blackboard key: %s", key)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.entries, key)
	return nil
}

// MergeFrom merges entries from another blackboard into this one.
// Used when sub-agents return results: their files_modified and decisions
// are merged into the parent's blackboard.
func (b *Blackboard) MergeFrom(other *Blackboard, sourceName string) {
	snap := other.Snapshot()
	for key, entries := range snap.Entries {
		for _, entry := range entries {
			b.Append(key, entry.Value, sourceName)
		}
	}
}

// Size returns the total number of entries across all keys.
func (b *Blackboard) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	count := 0
	for _, entries := range b.entries {
		count += len(entries)
	}
	return count
}

// Keys returns all keys that have at least one entry.
func (b *Blackboard) Keys() []BlackboardKey {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var keys []BlackboardKey
	for k, entries := range b.entries {
		if len(entries) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})
	return keys
}
