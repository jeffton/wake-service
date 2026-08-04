package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const openMeteoMarineURL = "https://marine-api.open-meteo.com/v1/marine"

type OpenMeteoMarineResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Hourly    struct {
		Time                  []int64    `json:"time"`
		WaveHeight            []*float64 `json:"wave_height"`
		WaveDirection         []*float64 `json:"wave_direction"`
		SeaSurfaceTemperature []*float64 `json:"sea_surface_temperature"`
	} `json:"hourly"`
}

func fetchOpenMeteoMarineData(client *http.Client, userAgent string, pos Position) (*OceanForecastData, []byte, error) {
	apiURL, err := url.Parse(openMeteoMarineURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse API URL: %w", err)
	}
	query := apiURL.Query()
	query.Set("latitude", strconv.FormatFloat(pos.Lat, 'f', 4, 64))
	query.Set("longitude", strconv.FormatFloat(pos.Lon, 'f', 4, 64))
	query.Set("hourly", "wave_height,wave_direction,sea_surface_temperature")
	query.Set("timeformat", "unixtime")
	query.Set("forecast_days", "10")
	query.Set("cell_selection", "sea")
	apiURL.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, body, &HTTPStatusError{StatusCode: resp.StatusCode, Status: resp.Status}
	}

	var payload OpenMeteoMarineResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, body, fmt.Errorf("decode response: %w", err)
	}

	data := &OceanForecastData{
		Position: &Coordinates{payload.Latitude, payload.Longitude},
	}
	var hasMarineData bool
	for i, timestamp := range payload.Hourly.Time {
		entry := OceanForecastEntry{
			Time:           timestamp,
			SeaTemperature: payload.Hourly.SeaSurfaceTemperature[i],
			WaveHeight:     payload.Hourly.WaveHeight[i],
			WaveDirection:  payload.Hourly.WaveDirection[i],
		}
		if entry.SeaTemperature != nil || entry.WaveHeight != nil || entry.WaveDirection != nil {
			hasMarineData = true
		}
		data.Timeseries = append(data.Timeseries, entry)
	}
	if !hasMarineData {
		return nil, body, fmt.Errorf("response contains no marine forecast data")
	}

	return data, nil, nil
}
