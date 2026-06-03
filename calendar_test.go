package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalendarPostRequiresFullAPIKey(t *testing.T) {
	server := NewServer(Options{
		DataDir: t.TempDir(),
		ApiKeys: []ApiKey{
			{Name: "weather", Key: "weather-key", Type: apiKeyTypeWeather},
			{Name: "full", Key: "full-key", Type: apiKeyTypeFull},
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/calendar", strings.NewReader(calendarPayloadJSON()))
	request.Header.Set(apiKeyHeader, "weather-key")
	response := httptest.NewRecorder()

	server.handleCalendar(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestCalendarPostReplacesStoredSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "calendar.json")
	server := NewServer(Options{
		DataDir: dir,
		ApiKeys: []ApiKey{{Name: "full", Key: "full-key", Type: apiKeyTypeFull}},
	})

	first := httptest.NewRequest(http.MethodPost, "/calendar", strings.NewReader(calendarPayloadJSON()))
	first.Header.Set(apiKeyHeader, "full-key")
	firstResponse := httptest.NewRecorder()
	server.handleCalendar(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("expected first status %d, got %d", http.StatusOK, firstResponse.Code)
	}

	second := httptest.NewRequest(http.MethodPost, "/calendar", strings.NewReader(strings.ReplaceAll(calendarPayloadJSON(), "Dentist", "Lunch")))
	second.Header.Set(apiKeyHeader, "full-key")
	secondResponse := httptest.NewRecorder()
	server.handleCalendar(secondResponse, second)
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("expected second status %d, got %d", http.StatusOK, secondResponse.Code)
	}

	payload, exists, err := loadCalendar(path)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected stored calendar")
	}
	if payload.Events[0].Title != "Lunch" {
		t.Fatalf("expected replacement event title Lunch, got %q", payload.Events[0].Title)
	}
}

func calendarPayloadJSON() string {
	return `{
  "syncedAt": "2026-06-02T20:00:00Z",
  "windowStart": "2026-06-02T20:00:00Z",
  "windowEnd": "2026-06-09T20:00:00Z",
  "calendars": [{"id":"cal-1","title":"Personal","sourceTitle":"Google","sourceType":"calDAV","colorHex":"#34A853"}],
  "events": [{
    "id":"event-1",
    "eventIdentifier":"event-1",
    "calendarItemIdentifier":"item-1",
    "calendarItemExternalIdentifier":"google-event-1",
    "calendarIdentifier":"cal-1",
    "calendarTitle":"Personal",
    "sourceTitle":"Google",
    "title":"Dentist",
    "startDate":"2026-06-03T09:00:00Z",
    "endDate":"2026-06-03T10:00:00Z",
    "isAllDay":false,
    "availability":"busy",
    "status":"confirmed",
    "attendees":[],
    "alarms":[],
    "recurrenceRules":[]
  }]
}`
}
