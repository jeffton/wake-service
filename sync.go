package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SyncState struct {
	LastWorkout    int64  `json:"lastWorkout"`
	LastWakeupDate string `json:"lastWakeupDate"`
}

func (s *Server) syncEvents(lastWorkout int64, awake *bool) error {
	state, exists, err := loadSyncState(s.options.SyncStatePath)
	if err != nil {
		return err
	}

	workoutTrigger, workoutState := evaluateWorkoutSyncState(state, exists, lastWorkout)
	wakeupTrigger, wakeupState := evaluateWakeupSyncState(state, awake, time.Now(), s.options.Cron.Wakeup.TriggerHour())

	nextState := state
	shouldUpdate := false
	var syncErrors []error

	if workoutTrigger {
		config := s.options.Cron.Workout
		if err := s.scheduleCron(config.Command, config.Prompt); err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("workout cron error: %w", err))
		} else {
			nextState.LastWorkout = workoutState.LastWorkout
			shouldUpdate = true
		}
	}
	if wakeupTrigger {
		config := s.options.Cron.Wakeup
		if err := s.scheduleCron(config.Command, config.Prompt); err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("wakeup cron error: %w", err))
		} else {
			nextState.LastWakeupDate = wakeupState.LastWakeupDate
			shouldUpdate = true
		}
	}

	if shouldUpdate {
		if err := writeSyncState(s.options.SyncStatePath, nextState); err != nil {
			syncErrors = append(syncErrors, err)
		}
	}

	return errors.Join(syncErrors...)
}

func evaluateWorkoutSyncState(state SyncState, exists bool, lastWorkout int64) (bool, SyncState) {
	if lastWorkout == 0 {
		return false, state
	}

	if exists && lastWorkout == state.LastWorkout {
		return false, state
	}

	state.LastWorkout = lastWorkout
	return true, state
}

func evaluateWakeupSyncState(state SyncState, awake *bool, now time.Time, triggerHour int) (bool, SyncState) {
	if awake == nil || !*awake || now.Hour() < triggerHour {
		return false, state
	}

	wakeupDate := now.Format("2006-01-02")
	if state.LastWakeupDate == wakeupDate {
		return false, state
	}

	state.LastWakeupDate = wakeupDate
	return true, state
}

func loadSyncState(path string) (SyncState, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SyncState{}, false, nil
		}
		return SyncState{}, false, fmt.Errorf("read sync state file: %w", err)
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return SyncState{}, false, fmt.Errorf("decode sync state file: %w", err)
	}

	return state, true, nil
}

func writeSyncState(path string, state SyncState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir sync state dir: %w", err)
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sync state: %w", err)
	}

	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return fmt.Errorf("write temp sync state file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp sync state file: %w", err)
	}

	return nil
}
