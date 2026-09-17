package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneState(t *testing.T) {
	const day = 86400
	now := int64(1000 * day)

	s := &State{Seen: map[string]int64{
		"old":      now - 40*day,
		"boundary": now - 30*day,
		"recent":   now - 5*day,
	}}

	pruned := pruneState(s, now, 30)

	if pruned != 1 {
		t.Fatalf("pruneState returned %d, want 1", pruned)
	}
	if _, ok := s.Seen["old"]; ok {
		t.Error("expected \"old\" entry to be pruned")
	}
	if _, ok := s.Seen["boundary"]; !ok {
		t.Error("expected \"boundary\" entry (exactly at cutoff) to survive")
	}
	if _, ok := s.Seen["recent"]; !ok {
		t.Error("expected \"recent\" entry to survive")
	}
}

func TestPruneStateNoneOld(t *testing.T) {
	now := int64(1000000)
	s := &State{Seen: map[string]int64{"a": now, "b": now - 10}}

	if pruned := pruneState(s, now, 30); pruned != 0 {
		t.Fatalf("pruneState returned %d, want 0", pruned)
	}
	if len(s.Seen) != 2 {
		t.Fatalf("expected both entries to survive, got %d", len(s.Seen))
	}
}

func TestSaveStateThenLoadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "state.json")

	want := &State{Seen: map[string]int64{"a": 1, "b": 2}}
	if err := saveState(path, want); err != nil {
		t.Fatalf("saveState: %v", err)
	}

	got, err := loadState(path)
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if len(got.Seen) != len(want.Seen) {
		t.Fatalf("loaded %d entries, want %d", len(got.Seen), len(want.Seen))
	}
	for id, ts := range want.Seen {
		if got.Seen[id] != ts {
			t.Errorf("entry %q = %d, want %d", id, got.Seen[id], ts)
		}
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("expected temp file to be renamed away, not left behind")
	}
}

func TestLoadStateMissingFile(t *testing.T) {
	s, err := loadState(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if s.Seen == nil || len(s.Seen) != 0 {
		t.Fatalf("expected empty, non-nil Seen map, got %#v", s.Seen)
	}
}
