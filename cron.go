package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func (s *Server) scheduleCron(command, prompt string) error {
	commandText := strings.TrimSpace(command)
	commandText = strings.ReplaceAll(commandText, "{prompt}", shellQuote(strings.TrimSpace(prompt)))

	cmd := exec.Command("/bin/sh", "-c", commandText)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cron command failed: %w, output: %s", err, string(output))
	}

	return nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
