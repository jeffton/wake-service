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
	UserAgent string      `json:"userAgent"`
	DataDir   string      `json:"dataDir"`
	ApiKeys   []ApiKey    `json:"apiKeys"`
	Cron      CronOptions `json:"cron"`
}

type ApiKey struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Type string `json:"type"`
}

type CronOptions struct {
	Workout CronJobOptions    `json:"workout"`
	Wakeup  WakeupCronOptions `json:"wakeup"`
}

type CronJobOptions struct {
	Command string `json:"command"`
	Prompt  string `json:"prompt"`
}

type WakeupCronOptions struct {
	Command string `json:"command"`
	Prompt  string `json:"prompt"`
	Hour    *int   `json:"hour"`
}

func (options WakeupCronOptions) TriggerHour() int {
	if options.Hour == nil {
		return 4
	}
	return *options.Hour
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

	if options.Cron.Workout.Command == "" {
		return Options{}, fmt.Errorf("cron.workout.command must be set in options file %s", path)
	}
	if options.Cron.Workout.Prompt == "" {
		return Options{}, fmt.Errorf("cron.workout.prompt must be set in options file %s", path)
	}
	if options.Cron.Wakeup.Command == "" {
		return Options{}, fmt.Errorf("cron.wakeup.command must be set in options file %s", path)
	}
	if options.Cron.Wakeup.Prompt == "" {
		return Options{}, fmt.Errorf("cron.wakeup.prompt must be set in options file %s", path)
	}
	if options.Cron.Wakeup.TriggerHour() < 0 || options.Cron.Wakeup.TriggerHour() > 23 {
		return Options{}, fmt.Errorf("cron.wakeup.hour must be between 0 and 23 in options file %s", path)
	}

	for _, key := range options.ApiKeys {
		if key.Type != apiKeyTypeWeather && key.Type != apiKeyTypeFull {
			return Options{}, fmt.Errorf("api key %s has invalid type %q (use weather or full)", key.Name, key.Type)
		}
	}

	if options.DataDir == "" {
		options.DataDir = filepath.Dir(path)
	}

	return options, nil
}

func (options Options) LocationPath() string {
	return filepath.Join(options.DataDir, "location.json")
}

func (options Options) CalendarPath() string {
	return filepath.Join(options.DataDir, "calendar.json")
}

func (options Options) SyncStatePath() string {
	return filepath.Join(options.DataDir, "sync-state.json")
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
