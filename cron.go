package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func (s *Server) scheduleCron(awake *bool) error {
	config := s.options.Cron
	prompt := strings.TrimSpace(config.Prompt)
	awakeText := "unknown"
	if awake != nil {
		if *awake {
			awakeText = "awake"
		} else {
			awakeText = "asleep"
		}
		prompt = fmt.Sprintf("%s\n\nUser awake state: %s.", prompt, awakeText)
	}

	commandText := strings.TrimSpace(config.Command)
	commandText = strings.ReplaceAll(commandText, "{prompt}", shellQuote(prompt))
	commandText = strings.ReplaceAll(commandText, "{awake}", shellQuote(awakeText))

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
