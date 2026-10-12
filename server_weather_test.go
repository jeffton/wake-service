package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeatherEndpointsRejectMissingWeatherAndRecover(t *testing.T) {
	for _, endpoint := range []string{"weather", "sync"} {
		for _, format := range []string{formatJSON, formatCompact} {
			for _, failure := range []struct {
				name   string
				status int
				body   string
			}{
				{"upstream", http.StatusServiceUnavailable, "YR temporarily unavailable"},
				{"empty", http.StatusOK, `{"properties":{"timeseries":[]}}`},
				{"api-error", http.StatusOK, `{"properties":{"meta":{"error":{"reason":"unavailable"}}}}`},
				{"invalid-time", http.StatusOK, `{"properties":{"timeseries":[{"time":"invalid"}]}}`},
				{"invalid-json", http.StatusOK, `invalid JSON`},
			} {
				t.Run(endpoint+"/"+format+"/"+failure.name, func(t *testing.T) {
					server := NewServer(Options{
						UserAgent: "Wake test",
						DataDir:   t.TempDir(),
						ApiKeys:   []ApiKey{{Key: "test", Type: apiKeyTypeFull}},
					})
					if _, err := writeLocation(server.options.LocationPath(), Position{Lat: 59, Lon: 10}, ""); err != nil {
						t.Fatal(err)
					}
					failed := true
					server.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						switch req.URL.Path {
						case "/weatherapi/locationforecast/2.0/complete":
							if failed {
								return testHTTPResponse(failure.status, failure.body), nil
							}
							return testHTTPResponse(http.StatusOK, `{"properties":{"timeseries":[{"time":"2026-10-12T06:00:00Z","data":{"instant":{"details":{"air_temperature":9,"wind_speed":5}},"next_1_hours":{"summary":{"symbol_code":"fair_day"}}}}]}}`), nil
						case "/weatherapi/oceanforecast/2.0/complete":
							return testHTTPResponse(http.StatusOK, `{"properties":{"timeseries":[{"time":"2026-10-12T06:00:00Z","data":{"instant":{"details":{"sea_water_temperature":14.6}}}}]}}`), nil
						default:
							return testHTTPResponse(http.StatusOK, `{"properties":{"timeseries":[]}}`), nil
						}
					})}
					request := func() *httptest.ResponseRecorder {
						var req *http.Request
						if endpoint == "sync" {
							req = httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"format":"`+format+`"}`))
						} else {
							req = httptest.NewRequest(http.MethodGet, "/weather?format="+format, nil)
						}
						req.Header.Set("X-Api-Key", "test")
						response := httptest.NewRecorder()
						if endpoint == "sync" {
							server.handleSync(response, req)
						} else {
							server.handleWeather(response, req)
						}
						return response
					}

					var logs bytes.Buffer
					originalOutput := log.Writer()
					log.SetOutput(&logs)
					defer log.SetOutput(originalOutput)
					response := request()
					var payload map[string]json.RawMessage
					if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
						t.Fatal(err)
					}
					if response.Code != http.StatusBadGateway || payload["error"] == nil || payload["forecast"] != nil {
						t.Fatalf("expected 502 error without forecast, got %d %s", response.Code, response.Body.String())
					}
					if !strings.Contains(logs.String(), "lat=59.0000 lon=10.0000") || !strings.Contains(logs.String(), "Weather unavailable") {
						t.Fatalf("missing failure context: %s", logs.String())
					}
					if failure.name == "upstream" && (!strings.Contains(logs.String(), "Locationforecast failed") || !strings.Contains(logs.String(), failure.body)) {
						t.Fatalf("upstream error not logged: %s", logs.String())
					}

					failed = false
					response = request()
					if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "forecast") || strings.Contains(response.Body.String(), `"error"`) {
						t.Fatalf("expected immediate recovery without cached failure, got %d %s", response.Code, response.Body.String())
					}
				})
			}
		}
	}
}
