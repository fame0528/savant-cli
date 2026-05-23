package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UsageRecord tracks how a skill has been used.
type UsageRecord struct {
	UseCount      int       `json:"use_count"`
	ViewCount     int       `json:"view_count"`
	PatchCount    int       `json:"patch_count"`
	LastUsedAt    time.Time `json:"last_used_at"`
	LastViewedAt  time.Time `json:"last_viewed_at"`
	LastPatchedAt time.Time `json:"last_patched_at"`
	State         string    `json:"state"` // "active", "stale", "archived"
	Pinned        bool      `json:"pinned"`
	CreatedBy     string    `json:"created_by"` // "agent", "user", "builtin"
}

// UsageStore manages the .usage.json sidecar file.
type UsageStore struct {
	path string
	mu   sync.RWMutex
	data map[string]*UsageRecord
}

// NewUsageStore creates or loads the usage store from disk.
func NewUsageStore(skillsDir string) *UsageStore {
	path := filepath.Join(skillsDir, ".usage.json")
	store := &UsageStore{
		path: path,
		data: make(map[string]*UsageRecord),
	}
	store.load()
	return store
}

// GetRecord returns the usage record for a skill, creating one if it doesn't exist.
func (s *UsageStore) GetRecord(name string) *UsageRecord {
	s.mu.RLock()
	rec, ok := s.data[name]
	s.mu.RUnlock()
	if ok {
		return rec
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock
	if rec, ok := s.data[name]; ok {
		return rec
	}
	rec = &UsageRecord{
		State:     "active",
		CreatedBy: "user",
	}
	s.data[name] = rec
	return rec
}

// SetCreatedBy marks a skill's provenance.
func (s *UsageStore) SetCreatedBy(name, provenance string) {
	rec := s.GetRecord(name)
	s.mu.Lock()
	rec.CreatedBy = provenance
	s.mu.Unlock()
	s.save()
}

// BumpView increments the view count for a skill.
func (s *UsageStore) BumpView(name string) {
	rec := s.GetRecord(name)
	s.mu.Lock()
	rec.ViewCount++
	rec.LastViewedAt = time.Now()
	s.mu.Unlock()
	s.save()
}

// BumpUse increments the use count for a skill.
func (s *UsageStore) BumpUse(name string) {
	rec := s.GetRecord(name)
	s.mu.Lock()
	rec.UseCount++
	rec.LastUsedAt = time.Now()
	rec.State = "active" // Reset to active on use
	s.mu.Unlock()
	s.save()
}

// BumpPatch increments the patch count for a skill.
func (s *UsageStore) BumpPatch(name string) {
	rec := s.GetRecord(name)
	s.mu.Lock()
	rec.PatchCount++
	rec.LastPatchedAt = time.Now()
	s.mu.Unlock()
	s.save()
}

// Pin marks a skill as pinned (exempt from auto-transitions and deletion).
func (s *UsageStore) Pin(name string) {
	rec := s.GetRecord(name)
	s.mu.Lock()
	rec.Pinned = true
	s.mu.Unlock()
	s.save()
}

// Unpin removes the pinned flag from a skill.
func (s *UsageStore) Unpin(name string) {
	rec := s.GetRecord(name)
	s.mu.Lock()
	rec.Pinned = false
	s.mu.Unlock()
	s.save()
}

// IsPinned returns whether a skill is pinned.
func (s *UsageStore) IsPinned(name string) bool {
	rec := s.GetRecord(name)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return rec.Pinned
}

// IsAgentCreated returns whether a skill was created by the agent.
func (s *UsageStore) IsAgentCreated(name string) bool {
	rec := s.GetRecord(name)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return rec.CreatedBy == "agent"
}

// GetState returns the lifecycle state of a skill.
func (s *UsageStore) GetState(name string) string {
	rec := s.GetRecord(name)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return rec.State
}

// SetState updates the lifecycle state of a skill.
func (s *UsageStore) SetState(name, state string) {
	rec := s.GetRecord(name)
	s.mu.Lock()
	rec.State = state
	s.mu.Unlock()
	s.save()
}

// AgentCreatedSkills returns all skills created by the agent.
func (s *UsageStore) AgentCreatedSkills() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var names []string
	for name, rec := range s.data {
		if rec.CreatedBy == "agent" {
			names = append(names, name)
		}
	}
	return names
}

// AllRecords returns all usage records.
func (s *UsageStore) AllRecords() map[string]*UsageRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*UsageRecord, len(s.data))
	for k, v := range s.data {
		result[k] = v
	}
	return result
}

// load reads the usage store from disk.
func (s *UsageStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return // File doesn't exist yet, that's fine
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	json.Unmarshal(data, &s.data)
}

// save writes the usage store to disk atomically.
func (s *UsageStore) save() {
	// Marshal current data (already holding lock from caller)
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return
	}

	// Atomic write: write to temp file, then rename
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return
	}
	os.Rename(tmpPath, s.path)
}

// SaveExternal saves the store (for use outside of locked methods).
func (s *UsageStore) SaveExternal() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.save()
}
