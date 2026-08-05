package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestFetchNowcastDataIgnoresPositionsOutsideCoverage(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		response := testHTTPResponse(http.StatusUnprocessableEntity, "outside coverage")
		response.Header.Set("X-ErrorClass", "Outsidegeographicarea")
		return response, nil
	})}

	data, rawBody, err := fetchNowcastData(client, "Wake test", Position{Lat: 37.754, Lon: 26.978})
	if err != nil {
		t.Fatalf("fetch nowcast data: %v", err)
	}
	if data != nil || rawBody != nil {
		t.Fatalf("data = %v, body = %q; want no nowcast outside coverage", data, rawBody)
	}
}

func TestNowcastOverridesCurrentLocationForecast(t *testing.T) {
	now := time.Date(2026, 8, 5, 5, 58, 0, 0, time.UTC)
	currentHour := now.Truncate(time.Hour).Unix()
	condition := "partly cloudy 40"
	temperature := 20.7
	windSpeed := 3.6
	windDirection := 140.0
	precipitationProbability := 12.3
	precipitationAmount := 0.0
	precipitation12Hours := 44.4
	forecasts := map[int64]*Forecast{
		currentHour: {
			TimeUnix:                 currentHour,
			Temperature:              &temperature,
			WindSpeed:                &windSpeed,
			WindDirection:            &windDirection,
			Condition:                &condition,
			Precipitation1Hour:       &precipitationProbability,
			PrecipitationAmount1Hour: &precipitationAmount,
			Precipitation12Hours:     &precipitation12Hours,
		},
	}
	weatherTimes := map[int64]bool{currentHour: true}

	var nowcast NowcastYrResponse
	if err := json.Unmarshal([]byte(`{
		"properties": {
			"meta": {"radar_coverage": "ok"},
			"timeseries": [{
				"time": "2026-08-05T06:00:00Z",
				"data": {
					"instant": {"details": {
						"air_temperature": 20.4,
						"precipitation_rate": 9.4,
						"wind_from_direction": 138.3,
						"wind_speed": 3.6
					}},
					"next_1_hours": {
						"summary": {"symbol_code": "heavyrainshowersandthunder_day"},
						"details": {"precipitation_amount": 6.1}
					}
				}
			}]
		}
	}`), &nowcast); err != nil {
		t.Fatalf("unmarshal nowcast: %v", err)
	}

	applyNowcast(forecasts, weatherTimes, &nowcast, now)
	got := forecasts[currentHour]
	if got.Condition == nil || *got.Condition != "thunder" {
		t.Fatalf("condition = %v, want thunder", got.Condition)
	}
	if got.Precipitation1Hour == nil || *got.Precipitation1Hour != 12.3 {
		t.Fatalf("precipitation probability = %v, want unchanged Locationforecast value 12.3", got.Precipitation1Hour)
	}
	if got.PrecipitationAmount1Hour == nil || *got.PrecipitationAmount1Hour != 6.1 {
		t.Fatalf("precipitation amount = %v, want 6.1", got.PrecipitationAmount1Hour)
	}
	if got.Precipitation12Hours == nil || *got.Precipitation12Hours != 100 {
		t.Fatalf("12-hour precipitation probability = %v, want 100", got.Precipitation12Hours)
	}
	if got.Temperature == nil || *got.Temperature != 20.4 {
		t.Fatalf("temperature = %v, want 20.4", got.Temperature)
	}
}

func TestDryNowcastPreservesTwelveHourLocationForecast(t *testing.T) {
	now := time.Date(2026, 8, 5, 5, 58, 0, 0, time.UTC)
	currentHour := now.Truncate(time.Hour).Unix()
	probability := 44.4
	forecasts := map[int64]*Forecast{currentHour: {TimeUnix: currentHour, Precipitation12Hours: &probability}}
	nowcast := &NowcastYrResponse{}
	nowcast.Properties.Meta.RadarCoverage = "ok"
	entry := NowcastTimeseriesEntry{Time: "2026-08-05T06:00:00Z"}
	rate := 0.0
	amount := 0.0
	entry.Data.Instant.Details.PrecipitationRate = &rate
	entry.Data.Next1Hours.Details.PrecipitationAmount = &amount
	entry.Data.Next1Hours.Summary.SymbolCode = "clearsky_day"
	nowcast.Properties.Timeseries = append(nowcast.Properties.Timeseries, entry)

	applyNowcast(forecasts, map[int64]bool{currentHour: true}, nowcast, now)
	if got := forecasts[currentHour].Precipitation12Hours; got == nil || *got != 44.4 {
		t.Fatalf("12-hour precipitation probability = %v, want preserved value 44.4", got)
	}
}

func TestStaleNowcastDoesNotOverrideForecast(t *testing.T) {
	now := time.Date(2026, 8, 5, 5, 58, 0, 0, time.UTC)
	currentHour := now.Truncate(time.Hour).Unix()
	condition := "clear"
	forecasts := map[int64]*Forecast{currentHour: {TimeUnix: currentHour, Condition: &condition}}
	nowcast := &NowcastYrResponse{}
	nowcast.Properties.Meta.RadarCoverage = "ok"
	entry := NowcastTimeseriesEntry{Time: "2026-08-05T05:30:00Z"}
	entry.Data.Next1Hours.Summary.SymbolCode = "rain"
	nowcast.Properties.Timeseries = append(nowcast.Properties.Timeseries, entry)

	applyNowcast(forecasts, map[int64]bool{currentHour: true}, nowcast, now)
	if got := *forecasts[currentHour].Condition; got != "clear" {
		t.Fatalf("condition = %q, want clear", got)
	}
}

func TestNowcastWithoutRadarCoverageDoesNotOverrideForecast(t *testing.T) {
	now := time.Date(2026, 8, 5, 5, 58, 0, 0, time.UTC)
	currentHour := now.Truncate(time.Hour).Unix()
	condition := "clear"
	forecasts := map[int64]*Forecast{currentHour: {TimeUnix: currentHour, Condition: &condition}}
	nowcast := &NowcastYrResponse{}
	nowcast.Properties.Meta.RadarCoverage = "no coverage"
	nowcast.Properties.Timeseries = append(nowcast.Properties.Timeseries, NowcastTimeseriesEntry{Time: "2026-08-05T06:00:00Z"})

	applyNowcast(forecasts, map[int64]bool{currentHour: true}, nowcast, now)
	if got := *forecasts[currentHour].Condition; got != "clear" {
		t.Fatalf("condition = %q, want clear", got)
	}
}
