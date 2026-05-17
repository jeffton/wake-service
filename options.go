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
	UserAgent     string      `json:"userAgent"`
	LocationPath  string      `json:"locationPath"`
	SyncStatePath string      `json:"syncStatePath"`
	ApiKeys       []ApiKey    `json:"apiKeys"`
	Cron          CronOptions `json:"cron"`
}

type ApiKey struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Type string `json:"type"`
}

type CronOptions struct {
	Command string `json:"command"`
	Prompt  string `json:"prompt"`
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

	if options.Cron.Command == "" {
		return Options{}, fmt.Errorf("cron.command must be set in options file %s", path)
	}
	if options.Cron.Prompt == "" {
		return Options{}, fmt.Errorf("cron.prompt must be set in options file %s", path)
	}

	for _, key := range options.ApiKeys {
		if key.Type != apiKeyTypeWeather && key.Type != apiKeyTypeFull {
			return Options{}, fmt.Errorf("api key %s has invalid type %q (use weather or full)", key.Name, key.Type)
		}
	}

	if options.LocationPath == "" {
		options.LocationPath = filepath.Join(filepath.Dir(path), "location.json")
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
