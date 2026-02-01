package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func (s *Server) scheduleOpenClaw() error {
	config := s.options.OpenClaw
	if config.URL == "" || config.Token == "" {
		return fmt.Errorf("openclaw is not configured")
	}

	text := strings.TrimSpace(config.Prompt)
	if text == "" {
		text = "The user has logged an activity with Garmin. Check Garmin stats and give feedback."
	}

	// Convert http:// to ws:// for CLI
	wsURL := config.URL
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	cmd := exec.Command("openclaw", "system", "event",
		"--text", text,
		"--mode", "now",
		"--url", wsURL,
		"--token", config.Token,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openclaw CLI failed: %w, output: %s", err, string(output))
	}

	return nil
}
