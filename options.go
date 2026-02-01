package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const optionsEnvVar = "WAKE_OPTIONS_PATH"

const (
	defaultLinuxOptionsPath  = "/etc/wake-service/options.json"
	defaultDarwinOptionsPath = "Library/Application Support/wake-service/options.json"
)

type Options struct {
	UserAgent       string          `json:"userAgent"`
	LocationLogPath string          `json:"locationLogPath"`
	SyncStatePath   string          `json:"syncStatePath"`
	ApiKeys         []ApiKey        `json:"apiKeys"`
	OpenClaw        OpenClawOptions `json:"openClaw"`
}

type ApiKey struct {
	Name             string `json:"name"`
	Key              string `json:"key"`
	AllowLocationLog bool   `json:"allowLocationLog"`
	AllowWorkout     bool   `json:"allowWorkout"`
}

type OpenClawOptions struct {
	URL          string `json:"url"`
	Token        string `json:"token"`
	DelayMinutes int    `json:"delayMinutes"`
	Prompt       string `json:"prompt"`
}

func loadOptions() (Options, error) {
	path := optionsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return Options{}, fmt.Errorf("read options file %s: %w", path, err)
	}

	var options Options
	if err := json.Unmarshal(data, &options); err != nil {
		return Options{}, fmt.Errorf("parse options file %s: %w", path, err)
	}

	if options.UserAgent == "" {
		return Options{}, fmt.Errorf("userAgent must be set in options file %s", path)
	}

	if len(options.ApiKeys) == 0 {
		return Options{}, fmt.Errorf("apiKeys must be set in options file %s", path)
	}

	if options.SyncStatePath == "" {
		options.SyncStatePath = filepath.Join(filepath.Dir(path), "sync-state.json")
	}

	return options, nil
}

func optionsPath() string {
	if value := os.Getenv(optionsEnvVar); value != "" {
		return value
	}

	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, defaultDarwinOptionsPath)
		}
	}

	return defaultLinuxOptionsPath
}
