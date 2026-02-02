package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func (s *Server) scheduleOpenClaw() error {
	config := s.options.OpenClaw

	text := strings.TrimSpace(config.Prompt)
	if text == "" {
		text = "The user has logged an activity with Garmin. Check Garmin stats and give feedback."
	}

	// Schedule via cron with delay (survives service restart)
	args := []string{"cron", "add",
		"--name", "Garmin workout ping",
		"--delete-after-run",
		"--system-event", text,
	}

	// Add delay if configured, otherwise immediate
	if config.DelayMinutes > 0 {
		args = append(args, "--at", fmt.Sprintf("%dm", config.DelayMinutes))
	} else {
		args = append(args, "--at", "0m")
	}

	cmd := exec.Command("openclaw", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openclaw CLI failed: %w, output: %s", err, string(output))
	}

	return nil
}
