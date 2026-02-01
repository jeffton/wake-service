package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type WorkoutSyncState struct {
	Date     int64 `json:"date"`
	Workouts int64 `json:"workouts"`
}

func parseSyncWorkoutParams(r *http.Request) (int64, int64, error) {
	dateStr := strings.TrimSpace(r.URL.Query().Get("date"))
	if dateStr == "" {
		return 0, 0, errors.New("date is required")
	}
	dateValue, err := strconv.ParseInt(dateStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid date value: %w", err)
	}

	workoutsStr := strings.TrimSpace(r.URL.Query().Get("workouts"))
	if workoutsStr == "" {
		return 0, 0, errors.New("workouts is required")
	}
	workouts, err := strconv.ParseInt(workoutsStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid workouts count: %w", err)
	}
	if workouts < 0 {
		return 0, 0, errors.New("workouts must be a positive number")
	}

	return workouts, dateValue, nil
}

func (s *Server) syncWorkouts(date int64, workouts int64) error {
	state, exists, err := loadWorkoutSyncState(s.options.SyncStatePath)
	if err != nil {
		return err
	}

	shouldTrigger, nextState, shouldUpdate := evaluateWorkoutSyncState(state, exists, date, workouts)

	if shouldTrigger {
		if err := s.scheduleOpenClaw(workouts); err != nil {
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

func evaluateWorkoutSyncState(state WorkoutSyncState, exists bool, date int64, workouts int64) (bool, WorkoutSyncState, bool) {
	shouldTrigger := false
	shouldUpdate := false

	if !exists {
		if workouts > 0 {
			shouldTrigger = true
		}
		return shouldTrigger, WorkoutSyncState{Date: date, Workouts: workouts}, true
	}

	switch {
	case date < state.Date:
		return false, state, false
	case date > state.Date:
		if workouts > 0 {
			shouldTrigger = true
		}
		return shouldTrigger, WorkoutSyncState{Date: date, Workouts: workouts}, true
	default:
		if workouts > state.Workouts {
			shouldUpdate = true
			state.Workouts = workouts
			shouldTrigger = true
		}
		return shouldTrigger, state, shouldUpdate
	}
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
