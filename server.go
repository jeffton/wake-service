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
	http.HandleFunc("/location", s.handleLocation)
	http.HandleFunc("/weather", s.handleWeather)
	http.HandleFunc("/sync", s.handleSync)
	http.HandleFunc("/calendar", s.handleCalendar)
}

func (s *Server) handleLocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	key, ok := authenticate(s.options, r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !key.IsFull() {
		writeJSONError(w, http.StatusForbidden, "location access not allowed for this api key")
		return
	}

	switch r.Method {
	case http.MethodGet:
		_, location, exists, err := loadLocation(s.options.LocationPath())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !exists {
			writeJSONError(w, http.StatusNotFound, "no stored location")
			return
		}
		writeJSON(w, http.StatusOK, location)
	case http.MethodPost:
		pos, precision, err := readLocationPost(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		location, err := writeLocation(s.options.LocationPath(), pos, precision)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, location)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCalendar(w http.ResponseWriter, r *http.Request) {
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
	if !key.IsFull() {
		writeJSONError(w, http.StatusForbidden, "calendar access not allowed for this api key")
		return
	}

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid calendar json: %v", err))
		return
	}
	payload, err := readCalendarPost(raw)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	response, err := writeCalendar(s.options.CalendarPath(), payload)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
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

	pos, err := s.weatherPosition(r, key)
	if err != nil {
		writeJSONError(w, statusForPositionError(err), err.Error())
		return
	}

	format := parseFormat(r)
	if format == "" {
		http.Error(w, "invalid format (use json or compact)", http.StatusBadRequest)
		return
	}

	response := s.weatherResponse(pos)
	s.writeWeatherResponse(w, format, response)
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
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
	if !key.IsFull() {
		writeJSONError(w, http.StatusForbidden, "sync not allowed for this api key")
		return
	}

	pos, err := s.storedPosition()
	if err != nil {
		writeJSONError(w, statusForPositionError(err), err.Error())
		return
	}

	request, err := readSyncPost(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := s.weatherResponse(pos)
	if err := s.syncEvents(request.LastWorkout, request.Awake); err != nil {
		appendResponseError(&response, fmt.Sprintf("cron error: %v", err))
	}

	s.writeWeatherResponse(w, request.Format, response)
}

func (s *Server) weatherResponse(pos Position) ApiResponseJSON {
	cacheKey := cacheKeyForPosition(pos)
	if cached, ok := s.cache.Get(cacheKey, cacheTTL); ok {
		return cached
	}

	oceanData, weatherData, errors := s.fetchForecasts(pos)
	response := buildForecastResponse(oceanData, weatherData, pos, errors)

	if response.Error == nil {
		s.cache.Set(cacheKey, response)
	}

	return response
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

func (s *Server) fetchForecasts(pos Position) (*OceanForecastData, *WeatherYrResponse, []string) {
	var (
		oceanData   *OceanForecastData
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

var (
	errStoredLocationNotAllowed = errors.New("stored location not allowed for this api key")
	errNoStoredLocation         = errors.New("position is required and no location is stored")
)

func (s *Server) weatherPosition(r *http.Request, key *AuthenticatedKey) (Position, error) {
	pos, provided, err := getPosition(r)
	if err != nil {
		return Position{}, err
	}
	if provided {
		return pos, nil
	}
	if !key.IsFull() {
		return Position{}, errStoredLocationNotAllowed
	}
	return s.storedPosition()
}

func (s *Server) storedPosition() (Position, error) {
	pos, _, exists, err := loadLocation(s.options.LocationPath())
	if err != nil {
		return Position{}, err
	}
	if !exists {
		return Position{}, errNoStoredLocation
	}
	return pos, nil
}

func getPosition(r *http.Request) (Position, bool, error) {
	latStr := strings.TrimSpace(r.URL.Query().Get("lat"))
	lonStr := strings.TrimSpace(r.URL.Query().Get("lon"))

	if latStr == "" && lonStr == "" {
		return Position{}, false, nil
	}
	if latStr == "" || lonStr == "" {
		return Position{}, false, errors.New("lat and lon are required")
	}

	pos, err := parsePosition(latStr, lonStr)
	if err != nil {
		return Position{}, false, err
	}
	return pos, true, nil
}

func readLocationPost(r *http.Request) (Position, string, error) {
	var payload struct {
		Lat       float64 `json:"lat"`
		Lon       float64 `json:"lon"`
		Precision string  `json:"precision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return Position{}, "", fmt.Errorf("invalid location json: %w", err)
	}
	pos, err := normalizePosition(Position{Lat: payload.Lat, Lon: payload.Lon})
	if err != nil {
		return Position{}, "", err
	}
	return pos, payload.Precision, nil
}

type SyncPostRequest struct {
	LastWorkout int64
	Awake       *bool
	Format      string
}

func readSyncPost(r *http.Request) (SyncPostRequest, error) {
	var payload struct {
		LastWorkout int64  `json:"lastWorkout"`
		Awake       *int   `json:"awake"`
		Format      string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return SyncPostRequest{}, fmt.Errorf("invalid sync json: %w", err)
	}

	format := parseFormatValue(payload.Format)
	if format == "" {
		return SyncPostRequest{}, errors.New("invalid format (use json or compact)")
	}

	var awake *bool
	if payload.Awake != nil {
		switch *payload.Awake {
		case 1:
			parsed := true
			awake = &parsed
		case 0:
			parsed := false
			awake = &parsed
		default:
			return SyncPostRequest{}, errors.New("invalid awake value (use 1 or 0)")
		}
	}

	return SyncPostRequest{LastWorkout: payload.LastWorkout, Awake: awake, Format: format}, nil
}

func parsePosition(latStr, lonStr string) (Position, error) {
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return Position{}, fmt.Errorf("invalid latitude: %w", err)
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return Position{}, fmt.Errorf("invalid longitude: %w", err)
	}
	return normalizePosition(Position{Lat: lat, Lon: lon})
}

func normalizePosition(pos Position) (Position, error) {
	if pos.Lat < -90 || pos.Lat > 90 {
		return Position{}, errors.New("latitude must be between -90 and 90")
	}
	if pos.Lon < -180 || pos.Lon > 180 {
		return Position{}, errors.New("longitude must be between -180 and 180")
	}
	return Position{Lat: roundCoordinate(pos.Lat), Lon: roundCoordinate(pos.Lon)}, nil
}

func roundCoordinate(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func cacheKeyForPosition(pos Position) string {
	return fmt.Sprintf("%.4f,%.4f", pos.Lat, pos.Lon)
}

func parseFormat(r *http.Request) string {
	return parseFormatValue(r.URL.Query().Get("format"))
}

func parseFormatValue(value string) string {
	format := strings.ToLower(strings.TrimSpace(value))
	if format == "" {
		return formatJSON
	}
	if format == formatJSON || format == formatCompact {
		return format
	}
	return ""
}

func statusForPositionError(err error) int {
	if errors.Is(err, errStoredLocationNotAllowed) {
		return http.StatusForbidden
	}
	if errors.Is(err, errNoStoredLocation) {
		return http.StatusBadRequest
	}
	return http.StatusBadRequest
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
