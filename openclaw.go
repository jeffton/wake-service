package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func (s *Server) scheduleOpenClaw() error {
	config := s.options.OpenClaw
	if config.Token == "" {
		return fmt.Errorf("openclaw is not configured (missing token)")
	}

	text := strings.TrimSpace(config.Prompt)
	if text == "" {
		text = "The user has logged an activity with Garmin. Check Garmin stats and give feedback."
	}

	args := []string{"system", "event",
		"--text", text,
		"--mode", "now",
		"--token", config.Token,
	}

	// Only add --url if explicitly configured
	if config.URL != "" {
		args = append(args, "--url", config.URL)
	}

	cmd := exec.Command("openclaw", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openclaw CLI failed: %w, output: %s", err, string(output))
	}

	return nil
}
