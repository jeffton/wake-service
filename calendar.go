package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CalendarSyncPayload struct {
	SyncedAt    time.Time             `json:"syncedAt"`
	WindowStart time.Time             `json:"windowStart"`
	WindowEnd   time.Time             `json:"windowEnd"`
	Calendars   []SyncedCalendar      `json:"calendars"`
	Events      []SyncedCalendarEvent `json:"events"`
}

type SyncedCalendar struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	SourceTitle string  `json:"sourceTitle"`
	SourceType  string  `json:"sourceType"`
	ColorHex    *string `json:"colorHex,omitempty"`
}

type SyncedCalendarEvent struct {
	ID                             string                      `json:"id"`
	EventIdentifier                string                      `json:"eventIdentifier"`
	CalendarItemIdentifier         string                      `json:"calendarItemIdentifier"`
	CalendarItemExternalIdentifier *string                     `json:"calendarItemExternalIdentifier,omitempty"`
	CalendarIdentifier             string                      `json:"calendarIdentifier"`
	CalendarTitle                  string                      `json:"calendarTitle"`
	SourceTitle                    string                      `json:"sourceTitle"`
	Title                          string                      `json:"title"`
	Location                       *string                     `json:"location,omitempty"`
	Notes                          *string                     `json:"notes,omitempty"`
	URL                            *string                     `json:"url,omitempty"`
	StartDate                      time.Time                   `json:"startDate"`
	EndDate                        time.Time                   `json:"endDate"`
	TimeZone                       *string                     `json:"timeZone,omitempty"`
	IsAllDay                       bool                        `json:"isAllDay"`
	Availability                   string                      `json:"availability"`
	Status                         string                      `json:"status"`
	Organizer                      *SyncedCalendarParticipant  `json:"organizer,omitempty"`
	Attendees                      []SyncedCalendarParticipant `json:"attendees"`
	Alarms                         []SyncedCalendarAlarm       `json:"alarms"`
	RecurrenceRules                []string                    `json:"recurrenceRules"`
	LastModifiedDate               *time.Time                  `json:"lastModifiedDate,omitempty"`
	CreationDate                   *time.Time                  `json:"creationDate,omitempty"`
}

type SyncedCalendarParticipant struct {
	Name   *string `json:"name,omitempty"`
	URL    *string `json:"url,omitempty"`
	Role   string  `json:"role"`
	Status string  `json:"status"`
	Type   string  `json:"type"`
}

type SyncedCalendarAlarm struct {
	RelativeOffset *float64   `json:"relativeOffset,omitempty"`
	AbsoluteDate   *time.Time `json:"absoluteDate,omitempty"`
}

type StoredCalendarResponse struct {
	OK         bool  `json:"ok"`
	Calendars  int   `json:"calendars"`
	Events     int   `json:"events"`
	ReceivedAt int64 `json:"receivedAt"`
}

func readCalendarPost(rbody json.RawMessage) (CalendarSyncPayload, error) {
	var payload CalendarSyncPayload
	if err := json.Unmarshal(rbody, &payload); err != nil {
		return CalendarSyncPayload{}, fmt.Errorf("invalid calendar json: %w", err)
	}
	return payload, nil
}

func loadCalendar(path string) (CalendarSyncPayload, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CalendarSyncPayload{}, false, nil
		}
		return CalendarSyncPayload{}, false, fmt.Errorf("read calendar file: %w", err)
	}

	var payload CalendarSyncPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return CalendarSyncPayload{}, false, fmt.Errorf("decode calendar file: %w", err)
	}
	return payload, true, nil
}

func writeCalendar(path string, payload CalendarSyncPayload) (StoredCalendarResponse, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoredCalendarResponse{}, fmt.Errorf("mkdir calendar dir: %w", err)
	}

	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return StoredCalendarResponse{}, fmt.Errorf("marshal calendar: %w", err)
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return StoredCalendarResponse{}, fmt.Errorf("write temp calendar file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return StoredCalendarResponse{}, fmt.Errorf("rename temp calendar file: %w", err)
	}

	return StoredCalendarResponse{OK: true, Calendars: len(payload.Calendars), Events: len(payload.Events), ReceivedAt: time.Now().Unix()}, nil
}
