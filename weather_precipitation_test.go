package main

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestHourlyPrecipitationInJSONForecast(t *testing.T) {
	var weather WeatherYrResponse
	err := json.Unmarshal([]byte(`{
		"geometry":{"coordinates":[10,59]},
		"properties":{"timeseries":[{
			"time":"2026-07-31T12:00:00Z",
			"data":{
				"instant":{"details":{}},
				"next_1_hours":{"summary":{"symbol_code":"rain"},"details":{
					"probability_of_precipitation":37,
					"precipitation_amount":0.4
				}}
			}
		}]}
	}`), &weather)
	if err != nil {
		t.Fatalf("unmarshal weather response: %v", err)
	}

	response := buildForecastResponseAt(nil, &weather, nil, Position{Lat: 59, Lon: 10}, nil, time.Date(2026, 7, 31, 12, 30, 0, 0, time.UTC))
	if len(response.Forecast) != 1 {
		t.Fatalf("got %d forecasts, want 1", len(response.Forecast))
	}

	forecast := response.Forecast[0]
	if forecast.Precipitation1Hour == nil || *forecast.Precipitation1Hour != 37 {
		t.Fatalf("precipitation1hour = %v, want 37", forecast.Precipitation1Hour)
	}
	if forecast.PrecipitationAmount1Hour == nil || *forecast.PrecipitationAmount1Hour != 0.4 {
		t.Fatalf("precipitationAmount1hour = %v, want 0.4", forecast.PrecipitationAmount1Hour)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal response JSON: %v", err)
	}
	forecastOutput := output["forecast"].([]any)[0].(map[string]any)
	if forecastOutput["precipitation1hour"] != 37.0 {
		t.Errorf("JSON precipitation1hour = %v, want 37", forecastOutput["precipitation1hour"])
	}
	if forecastOutput["precipitationAmount1hour"] != 0.4 {
		t.Errorf("JSON precipitationAmount1hour = %v, want 0.4", forecastOutput["precipitationAmount1hour"])
	}

	units := output["meta"].(map[string]any)["units"].(map[string]any)
	if units["precipitation1hour"] != "percent" {
		t.Errorf("precipitation1hour unit = %v, want percent", units["precipitation1hour"])
	}
	if units["precipitationAmount1hour"] != "mm" {
		t.Errorf("precipitationAmount1hour unit = %v, want mm", units["precipitationAmount1hour"])
	}
}

func TestUnavailableHourlyPrecipitationOmittedFromJSON(t *testing.T) {
	var weather WeatherYrResponse
	if err := json.Unmarshal([]byte(`{
		"properties":{"timeseries":[{
			"time":"2026-07-31T12:00:00Z",
			"data":{"instant":{"details":{}}}
		}]}
	}`), &weather); err != nil {
		t.Fatalf("unmarshal weather response: %v", err)
	}

	response := buildForecastResponseAt(nil, &weather, nil, Position{}, nil, time.Date(2026, 7, 31, 12, 30, 0, 0, time.UTC))
	encoded, err := json.Marshal(response.Forecast[0])
	if err != nil {
		t.Fatalf("marshal forecast: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatalf("unmarshal forecast JSON: %v", err)
	}
	if _, ok := output["precipitation1hour"]; ok {
		t.Error("precipitation1hour should be omitted when unavailable")
	}
	if _, ok := output["precipitationAmount1hour"]; ok {
		t.Error("precipitationAmount1hour should be omitted when unavailable")
	}
}

func TestPastForecastHoursAreOmitted(t *testing.T) {
	var weather WeatherYrResponse
	if err := json.Unmarshal([]byte(`{
		"properties":{"timeseries":[
			{"time":"2026-08-31T16:00:00Z","data":{"instant":{"details":{}}}},
			{"time":"2026-08-31T17:00:00Z","data":{"instant":{"details":{}}}}
		]}
	}`), &weather); err != nil {
		t.Fatalf("unmarshal weather response: %v", err)
	}

	response := buildForecastResponseAt(nil, &weather, nil, Position{}, nil, time.Date(2026, 8, 31, 17, 0, 1, 0, time.UTC))
	if len(response.Forecast) != 1 {
		t.Fatalf("got %d forecasts, want 1", len(response.Forecast))
	}
	if got := response.Forecast[0].TimeUnix; got != time.Date(2026, 8, 31, 17, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("forecast time = %d, want current hour", got)
	}
}

func TestActiveNowcastAppliesToFirstCurrentForecastAfterHourBoundary(t *testing.T) {
	var weather WeatherYrResponse
	if err := json.Unmarshal([]byte(`{
		"properties":{"timeseries":[
			{"time":"2026-08-31T16:00:00Z","data":{"instant":{"details":{}},"next_12_hours":{"details":{"probability_of_precipitation":55.4}}}},
			{"time":"2026-08-31T17:00:00Z","data":{"instant":{"details":{}},"next_12_hours":{"details":{"probability_of_precipitation":54.8}}}}
		]}
	}`), &weather); err != nil {
		t.Fatalf("unmarshal weather response: %v", err)
	}

	var nowcast NowcastYrResponse
	if err := json.Unmarshal([]byte(`{
		"properties":{"meta":{"radar_coverage":"ok"},"timeseries":[{
			"time":"2026-08-31T17:00:00Z",
			"data":{"instant":{"details":{"precipitation_rate":0.9}}}
		}]}
	}`), &nowcast); err != nil {
		t.Fatalf("unmarshal nowcast response: %v", err)
	}

	requestTime := time.Date(2026, 8, 31, 17, 0, 1, 0, time.UTC)
	response := buildForecastResponseAt(nil, &weather, &nowcast, Position{}, nil, requestTime)
	if len(response.Forecast) != 1 {
		t.Fatalf("got %d forecasts, want 1", len(response.Forecast))
	}
	forecast := response.Forecast[0]
	if forecast.TimeUnix != requestTime.Truncate(time.Hour).Unix() {
		t.Fatalf("first forecast time = %d, want current hour", forecast.TimeUnix)
	}
	if forecast.Precipitation12Hours == nil || *forecast.Precipitation12Hours != 100 {
		t.Fatalf("12-hour precipitation probability = %v, want 100", forecast.Precipitation12Hours)
	}
}

func TestHourlyPrecipitationDoesNotChangeCompactForecast(t *testing.T) {
	probability := 37.0
	amount := 0.4
	base := Forecast{TimeUnix: 1234}
	withHourlyPrecipitation := base
	withHourlyPrecipitation.Precipitation1Hour = &probability
	withHourlyPrecipitation.PrecipitationAmount1Hour = &amount

	got := forecastToCompactArray(withHourlyPrecipitation)
	want := forecastToCompactArray(base)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compact forecast changed:\n got: %#v\nwant: %#v", got, want)
	}
	if len(got) != ForecastEntrySize-1 {
		t.Fatalf("compact forecast length = %d, want %d", len(got), ForecastEntrySize-1)
	}
}
