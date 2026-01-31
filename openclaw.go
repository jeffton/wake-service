package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type WorkoutRequest struct {
	ActivityCount int64 `json:"activityCount"`
}

type workoutPayload struct {
	ActivityCount json.Number `json:"activityCount"`
}

func (s *Server) scheduleOpenClaw(activityCount int64) error {
	config := s.options.OpenClaw
	if config.URL == "" || config.Token == "" {
		return fmt.Errorf("openclaw is not configured")
	}
	if strings.TrimSpace(config.Prompt) == "" {
		config.Prompt = "The user has logged an activity with Garmin. Check Garmin stats and give feedback."
	}

	text := fmt.Sprintf("%s Activity count today: %d.", strings.TrimSpace(config.Prompt), activityCount)
	payload := openClawRequestPayload{
		Job: openClawJob{
			Name: "Garmin activity feedback",
			Schedule: openClawSchedule{
				Kind: "at",
				At:   time.Now().Add(time.Duration(config.DelayMinutes) * time.Minute).UnixMilli(),
			},
			SessionTarget: "main",
			WakeMode:      "now",
			Payload: openClawJobPayload{
				Kind: "systemEvent",
				Text: text,
			},
			DeleteAfterRun: true,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode openclaw payload: %w", err)
	}

	url := strings.TrimRight(config.URL, "/") + "/api/cron/add"
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create openclaw request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+config.Token)
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("openclaw request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("openclaw returned status %s", response.Status)
	}

	return nil
}

type openClawRequestPayload struct {
	Job openClawJob `json:"job"`
}

type openClawJob struct {
	Name           string             `json:"name"`
	Schedule       openClawSchedule   `json:"schedule"`
	SessionTarget  string             `json:"sessionTarget"`
	WakeMode       string             `json:"wakeMode"`
	Payload        openClawJobPayload `json:"payload"`
	DeleteAfterRun bool               `json:"deleteAfterRun"`
}

type openClawSchedule struct {
	Kind string `json:"kind"`
	At   int64  `json:"at"`
}

type openClawJobPayload struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}
