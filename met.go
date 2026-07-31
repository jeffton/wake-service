package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	oceanForecastURL   = "https://api.met.no/weatherapi/oceanforecast/2.0/complete"
	weatherForecastURL = "https://api.met.no/weatherapi/locationforecast/2.0/complete"
)

type OceanYrResponse struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Meta struct {
			Error map[string]interface{} `json:"error"`
		} `json:"meta"`
		Timeseries []struct {
			Time string `json:"time"`
			Data struct {
				Instant struct {
					Details struct {
						SeaWaterTemperature         float64 `json:"sea_water_temperature"`
						SeaSurfaceWaveHeight        float64 `json:"sea_surface_wave_height"`
						SeaSurfaceWaveFromDirection float64 `json:"sea_surface_wave_from_direction"`
					} `json:"details"`
				} `json:"instant"`
			} `json:"data"`
		} `json:"timeseries"`
	} `json:"properties"`
}

type WeatherYrResponse struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Meta struct {
			Error map[string]interface{} `json:"error"`
		} `json:"meta"`
		Timeseries []WeatherTimeseriesEntry `json:"timeseries"`
	} `json:"properties"`
}

type WeatherTimeseriesEntry struct {
	Time string `json:"time"`
	Data struct {
		Instant struct {
			Details struct {
				AirTemperature           float64  `json:"air_temperature"`
				CloudAreaFraction        float64  `json:"cloud_area_fraction"`
				CloudAreaFractionLow     *float64 `json:"cloud_area_fraction_low"`
				CloudAreaFractionMedium  *float64 `json:"cloud_area_fraction_medium"`
				CloudAreaFractionHigh    *float64 `json:"cloud_area_fraction_high"`
				WindFromDirection        float64  `json:"wind_from_direction"`
				WindSpeed                float64  `json:"wind_speed"`
				UltravioletIndexClearSky float64  `json:"ultraviolet_index_clear_sky"`
			} `json:"details"`
		} `json:"instant"`
		Next1Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
			Details struct {
				ProbabilityOfPrecipitation *float64 `json:"probability_of_precipitation"`
				PrecipitationAmount        *float64 `json:"precipitation_amount"`
			} `json:"details"`
		} `json:"next_1_hours"`
		Next12Hours struct {
			Summary struct {
				SymbolCode string `json:"symbol_code"`
			} `json:"summary"`
			Details struct {
				ProbabilityOfPrecipitation *float64 `json:"probability_of_precipitation"`
			} `json:"details"`
		} `json:"next_12_hours"`
	} `json:"data"`
}

func fetchOceanData(client *http.Client, userAgent string, pos Position) (*OceanYrResponse, []byte, error) {
	apiURL := fmt.Sprintf("%s?lat=%.4f&lon=%.4f", oceanForecastURL, pos.Lat, pos.Lon)
	return fetchYrData[OceanYrResponse](client, userAgent, apiURL)
}

func fetchWeatherData(client *http.Client, userAgent string, pos Position) (*WeatherYrResponse, []byte, error) {
	apiURL := fmt.Sprintf("%s?lat=%.4f&lon=%.4f", weatherForecastURL, pos.Lat, pos.Lon)
	return fetchYrData[WeatherYrResponse](client, userAgent, apiURL)
}

func fetchYrData[T any](client *http.Client, userAgent, apiURL string) (*T, []byte, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
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

	var payload T
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, body, fmt.Errorf("decode response: %w", err)
	}

	return &payload, nil, nil
}

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}
