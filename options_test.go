package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOptionsDefaultsDataDirToOptionsDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "options.json")
	body := `{
  "userAgent": "Wake test",
  "apiKeys": [{"name":"full","key":"full-key","type":"full"}],
  "cron": {
    "workout": {"command":"true","prompt":"workout"},
    "wakeup": {"command":"true","prompt":"wakeup"}
  }
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(optionsEnvVar, path)

	options, err := loadOptions()
	if err != nil {
		t.Fatal(err)
	}

	if options.DataDir != dir {
		t.Fatalf("expected dataDir %q, got %q", dir, options.DataDir)
	}
	if options.LocationPath() != filepath.Join(dir, "location.json") {
		t.Fatalf("unexpected location path %q", options.LocationPath())
	}
	if options.CalendarPath() != filepath.Join(dir, "calendar.json") {
		t.Fatalf("unexpected calendar path %q", options.CalendarPath())
	}
	if options.SyncStatePath() != filepath.Join(dir, "sync-state.json") {
		t.Fatalf("unexpected sync state path %q", options.SyncStatePath())
	}
}

func TestLoadOptionsUsesConfiguredDataDir(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	path := filepath.Join(dir, "options.json")
	body := `{
  "userAgent": "Wake test",
  "dataDir": "` + dataDir + `",
  "apiKeys": [{"name":"full","key":"full-key","type":"full"}],
  "cron": {
    "workout": {"command":"true","prompt":"workout"},
    "wakeup": {"command":"true","prompt":"wakeup"}
  }
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(optionsEnvVar, path)

	options, err := loadOptions()
	if err != nil {
		t.Fatal(err)
	}

	if options.DataDir != dataDir {
		t.Fatalf("expected dataDir %q, got %q", dataDir, options.DataDir)
	}
	if options.LocationPath() != filepath.Join(dataDir, "location.json") {
		t.Fatalf("unexpected location path %q", options.LocationPath())
	}
}
