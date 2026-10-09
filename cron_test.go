package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatty2CronCommandPreservesPrompt(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "args")
	cli := filepath.Join(dir, "batty2")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CAPTURE\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("CAPTURE", capture)
	prompt := "David's activity; $(touch /should-not-exist)"
	s := &Server{}
	command := "batty2 cron add --workspace roy --prompt {prompt} --model openai-codex/gpt-6.1-sol --thinking medium --delivery direct --in 3m --session daily-detached --daily-context chat-only"
	if err := s.scheduleCron(command, prompt); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{"cron", "add", "--workspace", "roy", "--prompt", prompt, "--model", "openai-codex/gpt-6.1-sol", "--thinking", "medium", "--delivery", "direct", "--in", "3m", "--session", "daily-detached", "--daily-context", "chat-only", ""}, "\n")
	if string(data) != want {
		t.Fatalf("arguments = %q, want %q", data, want)
	}
}
