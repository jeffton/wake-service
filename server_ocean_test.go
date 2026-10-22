package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeatherPreservesLandForecastWhenMarineDataIsUnavailableOrFails(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		wantError string
	}{
		{
			name:   "inland-no-data",
			status: http.StatusOK,
			body:   `{"type":"Feature","geometry":{"type":"Point","coordinates":[15.592,56.4614]},"properties":{"meta":{"updated_at":"2026-10-22T15:01:00Z","error":"no data at the given location","units":{}},"timeseries":[]}}`,
		},
		{
			name:      "other-string-error",
			status:    http.StatusOK,
			body:      `{"properties":{"meta":{"error":"model unavailable"},"timeseries":[]}}`,
			wantError: "ocean API error: model unavailable",
		},
		{
			name:      "structured-error",
			status:    http.StatusOK,
			body:      `{"properties":{"meta":{"error":{"reason":"model unavailable"}},"timeseries":[]}}`,
			wantError: "ocean API error: map[reason:model unavailable]",
		},
		{
			name:      "inconsistent-no-data",
			status:    http.StatusOK,
			body:      `{"properties":{"meta":{"error":"no data at the given location"},"timeseries":[{"time":"2026-10-22T15:00:00Z"}]}}`,
			wantError: "ocean API error: no data at the given location",
		},
		{
			name:      "invalid-json",
			status:    http.StatusOK,
			body:      `invalid JSON`,
			wantError: "Error getting ocean data: decode response:",
		},
		{
			name:      "http-error",
			status:    http.StatusServiceUnavailable,
			body:      `upstream failure`,
			wantError: "Error getting ocean data: unexpected HTTP status:",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer(Options{
				UserAgent: "Wake test",
				DataDir:   t.TempDir(),
				ApiKeys:   []ApiKey{{Key: "test", Type: apiKeyTypeFull}},
			})
			server.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "api.met.no" {
					t.Fatalf("unexpected marine fallback: %s", req.URL)
				}
				switch req.URL.Path {
				case "/weatherapi/oceanforecast/2.0/complete":
					return testHTTPResponse(tc.status, tc.body), nil
				case "/weatherapi/locationforecast/2.0/complete":
					return testHTTPResponse(http.StatusOK, `{"properties":{"timeseries":[{"time":"2026-10-22T15:00:00Z","data":{"instant":{"details":{"air_temperature":10.4,"wind_speed":2.1}},"next_1_hours":{"summary":{"symbol_code":"lightrain"}}}}]}}`), nil
				default:
					return testHTTPResponse(http.StatusOK, `{"properties":{"timeseries":[]}}`), nil
				}
			})}

			req := httptest.NewRequest(http.MethodGet, "/weather?lat=56.4614&lon=15.5920", nil)
			req.Header.Set("X-Api-Key", "test")
			response := httptest.NewRecorder()
			server.handleWeather(response, req)
			var payload ApiResponseJSON
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || len(payload.Forecast) != 1 {
				t.Fatalf("land forecast lost: %d %s", response.Code, response.Body.String())
			}
			if payload.Forecast[0].Temperature == nil || *payload.Forecast[0].Temperature != 10.4 || payload.Forecast[0].SeaTemperature != nil || payload.OceanForecastPosition != nil {
				t.Fatalf("unexpected land/marine values: %s", response.Body.String())
			}
			if tc.wantError == "" {
				if payload.Error != nil {
					t.Fatalf("unexpected error for inland no-data response: %s", payload.Error)
				}
			} else if message, ok := payload.Error.(string); !ok || !strings.Contains(message, tc.wantError) {
				t.Fatalf("error = %q, want %q", payload.Error, tc.wantError)
			}
		})
	}
}
