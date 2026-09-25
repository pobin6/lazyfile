package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	want := State{
		Entries:       []Entry{{Path: "/tmp/example", Name: "example"}},
		SelectedIndex: 0,
	}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0] != want.Entries[0] || got.SelectedIndex != want.SelectedIndex {
		t.Fatalf("state = %+v, want %+v", got, want)
	}
	if _, err := os.Stat(filepath.Join(configDir, "lazyfile", "entries.json")); err != nil {
		t.Fatal(err)
	}
}
