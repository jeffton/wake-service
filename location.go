package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type StoredLocation struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Precision string  `json:"precision"`
	Ts        int64   `json:"ts"`
}

func loadLocation(path string) (Position, StoredLocation, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Position{}, StoredLocation{}, false, nil
		}
		return Position{}, StoredLocation{}, false, fmt.Errorf("read location file: %w", err)
	}

	var location StoredLocation
	if err := json.Unmarshal(data, &location); err != nil {
		return Position{}, StoredLocation{}, false, fmt.Errorf("decode location file: %w", err)
	}

	pos := Position{Lat: roundCoordinate(location.Lat), Lon: roundCoordinate(location.Lon)}
	location.Lat = pos.Lat
	location.Lon = pos.Lon
	return pos, location, true, nil
}

func writeLocation(path string, pos Position, precision string) (StoredLocation, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoredLocation{}, fmt.Errorf("mkdir location dir: %w", err)
	}

	location := StoredLocation{Lat: pos.Lat, Lon: pos.Lon, Precision: precision, Ts: time.Now().Unix()}
	body, err := json.MarshalIndent(location, "", "  ")
	if err != nil {
		return StoredLocation{}, fmt.Errorf("marshal location: %w", err)
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return StoredLocation{}, fmt.Errorf("write temp location file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return StoredLocation{}, fmt.Errorf("rename temp location file: %w", err)
	}

	return location, nil
}
