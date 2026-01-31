package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultLat = 55.7122
	defaultLon = 12.5890
)

const cacheTTL = time.Hour

const (
	formatJSON    = "json"
	formatCompact = "compact"
)

type Server struct {
	options    Options
	cache      *ForecastCache
	httpClient *http.Client
}

func NewServer(options Options) *Server {
	return &Server{
		options:    options,
		cache:      NewForecastCache(),
		httpClient: defaultHTTPClient(),
	}
}

func (s *Server) routes() {
	http.HandleFunc("/weather", s.handleWeather)
	http.HandleFunc("/workout", s.handleWorkout)
}

func (s *Server) handleWeather(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	key, ok := authenticate(s.options, r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	pos, err := getPosition(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	format := parseFormat(r)
	if format == "" {
		http.Error(w, "invalid format (use json or compact)", http.StatusBadRequest)
		return
	}

	cacheKey := cacheKeyForPosition(pos)
	if cached, ok := s.cache.Get(cacheKey, cacheTTL); ok {
		s.writeWeatherResponse(w, format, cached)
		return
	}

	if key.ApiKey.AllowLocationLog {
		if err := writeLocation(s.options.LocationLogPath, pos); err != nil {
			log.Printf("Failed to write location: %v", err)
		}
	}

	oceanData, weatherData, errors := s.fetchForecasts(pos)
	response := buildForecastResponse(oceanData, weatherData, pos, errors)

	if response.Error == nil {
		s.cache.Set(cacheKey, response)
	}

	s.writeWeatherResponse(w, format, response)
}

func (s *Server) writeWeatherResponse(w http.ResponseWriter, format string, response ApiResponseJSON) {
	if format == formatCompact {
		compact := buildCompactResponse(response)
		if err := json.NewEncoder(w).Encode(compact); err != nil {
			log.Printf("Failed to encode compact response: %v", err)
		}
		return
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *Server) fetchForecasts(pos Position) (*OceanYrResponse, *WeatherYrResponse, []string) {
	var (
		oceanData   *OceanYrResponse
		weatherData *WeatherYrResponse
		errors      []string
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		data, rawBody, err := fetchOceanData(s.httpClient, s.options.UserAgent, pos)
		if err != nil {
			errMsg := fmt.Sprintf("Error getting ocean data: %v", err)
			if rawBody != nil {
				errMsg = fmt.Sprintf("%s, body: %s", errMsg, string(rawBody))
			}
			mu.Lock()
			errors = append(errors, errMsg)
			mu.Unlock()
			return
		}
		oceanData = data
	}()

	go func() {
		defer wg.Done()
		data, rawBody, err := fetchWeatherData(s.httpClient, s.options.UserAgent, pos)
		if err != nil {
			errMsg := fmt.Sprintf("Error getting weather data: %v", err)
			if rawBody != nil {
				errMsg = fmt.Sprintf("%s, body: %s", errMsg, string(rawBody))
			}
			mu.Lock()
			errors = append(errors, errMsg)
			mu.Unlock()
			return
		}
		weatherData = data
	}()

	wg.Wait()

	return oceanData, weatherData, errors
}

func (s *Server) handleWorkout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	key, ok := authenticate(s.options, r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !key.ApiKey.AllowWorkout {
		writeJSONError(w, http.StatusForbidden, "workout not allowed for this api key")
		return
	}

	payload, err := decodeWorkoutRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.scheduleOpenClaw(payload.ActivityCount); err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func decodeWorkoutRequest(r *http.Request) (WorkoutRequest, error) {
	var payload workoutPayload
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return WorkoutRequest{}, fmt.Errorf("invalid json: %w", err)
	}
	if payload.ActivityCount == "" {
		return WorkoutRequest{}, errors.New("activityCount is required")
	}
	count, err := payload.ActivityCount.Int64()
	if err != nil {
		return WorkoutRequest{}, errors.New("activityCount must be a number")
	}
	if count < 0 {
		return WorkoutRequest{}, errors.New("activityCount must be a positive number")
	}
	return WorkoutRequest{ActivityCount: count}, nil
}

func getPosition(r *http.Request) (Position, error) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	lat, err := parseOrDefault(latStr, defaultLat)
	if err != nil {
		return Position{}, fmt.Errorf("invalid latitude: %w", err)
	}
	lon, err := parseOrDefault(lonStr, defaultLon)
	if err != nil {
		return Position{}, fmt.Errorf("invalid longitude: %w", err)
	}

	lat = roundCoordinate(lat)
	lon = roundCoordinate(lon)

	return Position{Lat: lat, Lon: lon}, nil
}

func parseOrDefault(value string, fallback float64) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(value, 64)
}

func roundCoordinate(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func cacheKeyForPosition(pos Position) string {
	return fmt.Sprintf("%.4f,%.4f", pos.Lat, pos.Lon)
}

func parseFormat(r *http.Request) string {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		return formatJSON
	}
	if format == formatJSON || format == formatCompact {
		return format
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Failed to encode json response: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": message})
}
