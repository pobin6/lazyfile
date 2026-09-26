package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Entry struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	CurrentPath string `json:"current_path,omitempty"`
}

type Collection struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`
}

type State struct {
	Entries            []Entry      `json:"entries"`
	SelectedIndex      int          `json:"selected_index"`
	Collections        []Collection `json:"collections,omitempty"`
	SelectedCollection int          `json:"selected_collection,omitempty"`
	CollectionPage     bool         `json:"collection_page,omitempty"`
}

func Load() (State, error) {
	path, err := filePath()
	if err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode state: %w", err)
	}
	return state, nil
}

func Save(state State) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}

func filePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(configDir, "lazyfile", "entries.json"), nil
}
