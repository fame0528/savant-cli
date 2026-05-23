package pet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveFile returns the default path for pet state persistence.
func SaveFile(dir string) string {
	return filepath.Join(dir, "pet.json")
}

// Save persists the pet state to a JSON file.
func (p *Pet) Save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create pet dir: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pet: %w", err)
	}
	path := SaveFile(dir)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write pet: %w", err)
	}
	return nil
}

// LoadPet loads a pet from a JSON file, or returns nil if not found.
func LoadPet(dir string) *Pet {
	path := SaveFile(dir)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var p Pet
	if err := json.Unmarshal(data, &p); err != nil {
		return nil
	}
	return &p
}
