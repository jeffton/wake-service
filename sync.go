package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type WorkoutSyncState struct {
	LastWorkout int64 `json:"lastWorkout"`
}

func (s *Server) syncWorkouts(lastWorkout int64, awake *bool) error {
	state, exists, err := loadWorkoutSyncState(s.options.SyncStatePath)
	if err != nil {
		return err
	}

	shouldTrigger, nextState, shouldUpdate := evaluateWorkoutSyncState(state, exists, lastWorkout)

	if shouldTrigger {
		if err := s.scheduleCron(awake); err != nil {
			return err
		}
	}

	if shouldUpdate {
		if err := writeWorkoutSyncState(s.options.SyncStatePath, nextState); err != nil {
			return err
		}
	}

	return nil
}

func evaluateWorkoutSyncState(state WorkoutSyncState, exists bool, lastWorkout int64) (bool, WorkoutSyncState, bool) {
	if lastWorkout == 0 {
		return false, state, false
	}

	if !exists {
		return true, WorkoutSyncState{LastWorkout: lastWorkout}, true
	}

	if lastWorkout == state.LastWorkout {
		return false, state, false
	}

	return true, WorkoutSyncState{LastWorkout: lastWorkout}, true
}

func loadWorkoutSyncState(path string) (WorkoutSyncState, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return WorkoutSyncState{}, false, nil
		}
		return WorkoutSyncState{}, false, fmt.Errorf("read sync state file: %w", err)
	}

	var state WorkoutSyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return WorkoutSyncState{}, false, fmt.Errorf("decode sync state file: %w", err)
	}

	return state, true, nil
}

func writeWorkoutSyncState(path string, state WorkoutSyncState) error {
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
