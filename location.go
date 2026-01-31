package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LocationLog struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Ts  int64   `json:"ts"`
}

func writeLocation(logPath string, pos Position) error {
	if logPath == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("mkdir location dir: %w", err)
	}

	payload := LocationLog{Lat: pos.Lat, Lon: pos.Lon, Ts: time.Now().Unix()}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal location: %w", err)
	}

	tmp := fmt.Sprintf("%s.tmp.%d", logPath, time.Now().UnixNano())
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return fmt.Errorf("write temp location file: %w", err)
	}
	if err := os.Rename(tmp, logPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp location file: %w", err)
	}

	return nil
}
