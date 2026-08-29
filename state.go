package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State records which item IDs we've already reported, plus when, so a
// stale entry could later be pruned (not implemented yet).
type State struct {
	Seen map[string]int64 `json:"seen"`
}

func loadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &State{Seen: map[string]int64{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Seen == nil {
		s.Seen = map[string]int64{}
	}
	return &s, nil
}

// saveState writes via a temp file + rename so a crash or a second
// process running concurrently can't leave a half-written state file.
func saveState(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
