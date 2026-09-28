package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const locationHistoryWindow = 24 * time.Hour

var locationWriteMu sync.Mutex

type StoredLocation struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Precision string  `json:"precision"`
	Ts        int64   `json:"ts"`
}

type LocationHistory struct {
	Locations []StoredLocation `json:"locations"`
}

func loadLocationHistory(path string) (LocationHistory, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return LocationHistory{}, nil
	}
	if err != nil {
		return LocationHistory{}, fmt.Errorf("read location file: %w", err)
	}

	// Migrate the original single-location file on the next POST.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return LocationHistory{}, fmt.Errorf("decode location file: %w", err)
	}
	if _, legacy := fields["lat"]; legacy {
		var location StoredLocation
		if err := json.Unmarshal(data, &location); err != nil {
			return LocationHistory{}, fmt.Errorf("decode location file: %w", err)
		}
		return LocationHistory{Locations: []StoredLocation{location}}, nil
	}
	var history LocationHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return LocationHistory{}, fmt.Errorf("decode location file: %w", err)
	}
	return history, nil
}

func loadLocation(path string) (Position, StoredLocation, bool, error) {
	history, err := loadLocationHistory(path)
	if err != nil {
		return Position{}, StoredLocation{}, false, err
	}
	if len(history.Locations) == 0 {
		return Position{}, StoredLocation{}, false, nil
	}
	location := history.Locations[len(history.Locations)-1]
	pos := Position{Lat: roundCoordinate(location.Lat), Lon: roundCoordinate(location.Lon)}
	location.Lat = pos.Lat
	location.Lon = pos.Lon
	return pos, location, true, nil
}

func writeLocation(path string, pos Position, precision string) (StoredLocation, error) {
	locationWriteMu.Lock()
	defer locationWriteMu.Unlock()

	history, err := loadLocationHistory(path)
	if err != nil {
		return StoredLocation{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoredLocation{}, fmt.Errorf("mkdir location dir: %w", err)
	}

	now := time.Now()
	location := StoredLocation{Lat: pos.Lat, Lon: pos.Lon, Precision: precision, Ts: now.Unix()}
	cutoff := now.Add(-locationHistoryWindow).Unix()
	locations := make([]StoredLocation, 0, len(history.Locations)+1)
	for _, entry := range history.Locations {
		if entry.Ts >= cutoff {
			locations = append(locations, entry)
		}
	}
	history.Locations = append(locations, location)
	body, err := json.MarshalIndent(history, "", "  ")
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
