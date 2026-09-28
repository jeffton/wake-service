package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLocationHistoryMigrationAndPruning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "location.json")
	old := StoredLocation{Lat: 55.1, Lon: 12.1, Precision: "old", Ts: time.Now().Add(-25 * time.Hour).Unix()}
	legacy, _ := json.Marshal(old)
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	_, stored, exists, err := loadLocation(path)
	if err != nil || !exists || stored != old {
		t.Fatalf("legacy location: %+v, exists=%v, err=%v", stored, exists, err)
	}

	first, err := writeLocation(path, Position{Lat: 55.2, Lon: 12.2}, "reduced")
	if err != nil {
		t.Fatal(err)
	}
	second, err := writeLocation(path, Position{Lat: 55.3, Lon: 12.3}, "precise")
	if err != nil {
		t.Fatal(err)
	}
	history, err := loadLocationHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Locations) != 2 || history.Locations[0] != first || history.Locations[1] != second {
		t.Fatalf("unexpected history: %+v", history)
	}
	_, latest, exists, err := loadLocation(path)
	if err != nil || !exists || latest != second {
		t.Fatalf("latest location: %+v, exists=%v, err=%v", latest, exists, err)
	}
}

func TestLocationHistoryRetainsRecentLegacyEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "location.json")
	legacy := StoredLocation{Lat: 55.1, Lon: 12.1, Ts: time.Now().Add(-time.Hour).Unix()}
	body, _ := json.Marshal(legacy)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := writeLocation(path, Position{Lat: 55.2, Lon: 12.2}, "reduced"); err != nil {
		t.Fatal(err)
	}
	history, err := loadLocationHistory(path)
	if err != nil || len(history.Locations) != 2 || history.Locations[0] != legacy {
		t.Fatalf("migration: %+v, err=%v", history, err)
	}
}

func TestLocationAPIExposesOnlyLatest(t *testing.T) {
	server := NewServer(Options{DataDir: t.TempDir(), ApiKeys: []ApiKey{{Key: "secret", Type: apiKeyTypeFull}}})
	for i, body := range []string{
		`{"lat":55.1,"lon":12.1,"precision":"reduced"}`,
		`{"lat":55.2,"lon":12.2,"precision":"precise"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/location", strings.NewReader(body))
		req.Header.Set("X-Api-Key", "secret")
		response := httptest.NewRecorder()
		server.handleLocation(response, req)
		assertLocationResponse(t, response, 55.1+float64(i)/10)
	}
	req := httptest.NewRequest(http.MethodGet, "/location", nil)
	req.Header.Set("X-Api-Key", "secret")
	response := httptest.NewRecorder()
	server.handleLocation(response, req)
	assertLocationResponse(t, response, 55.2)
}

func assertLocationResponse(t *testing.T, response *httptest.ResponseRecorder, expectedLat float64) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if _, hasHistory := payload["locations"]; hasHistory {
		t.Fatalf("API leaked history: %+v", payload)
	}
	if payload["lat"] != expectedLat || payload["lon"] == nil || payload["precision"] == nil || payload["ts"] == nil || len(payload) != 4 {
		t.Fatalf("unexpected response: %+v", payload)
	}
}

func TestConcurrentLocationWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "location.json")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := writeLocation(path, Position{Lat: 55, Lon: 12}, "reduced"); err != nil {
				t.Errorf("write location: %v", err)
			}
		}()
	}
	wg.Wait()
	history, err := loadLocationHistory(path)
	if err != nil || len(history.Locations) != 20 {
		t.Fatalf("concurrent writes: %d entries, err=%v", len(history.Locations), err)
	}
}
