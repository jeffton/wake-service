package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	forecastEntryLimit = 24
)

const (
	ForecastIdxTime = iota
	ForecastIdxSeaTemperature
	ForecastIdxWaveHeight
	ForecastIdxWaveDirection
	ForecastIdxTemperature
	ForecastIdxWindSpeed
	ForecastIdxWindDirection
	ForecastIdxCloudCover
	ForecastIdxCondition
	ForecastIdxUvIndex
	ForecastIdxPrecipitation
	ForecastEntrySize
)

func buildForecastResponse(oceanData *OceanYrResponse, weatherData *WeatherYrResponse, requestPos Position, errors []string) ApiResponseJSON {
	response := ApiResponseJSON{
		RequestPosition: Coordinates{requestPos.Lat, requestPos.Lon},
		RequestTime:     time.Now().Unix(),
	}

	if len(errors) > 0 {
		response.Error = strings.Join(errors, "; ")
	}

	forecasts := make(map[int64]*Forecast)

	if oceanData != nil {
		if len(oceanData.Geometry.Coordinates) >= 2 {
			response.OceanForecastPosition = &Coordinates{oceanData.Geometry.Coordinates[1], oceanData.Geometry.Coordinates[0]}
		}

		if oceanData.Properties.Meta.Error != nil {
			appendResponseError(&response, fmt.Sprintf("ocean API error: %v", oceanData.Properties.Meta.Error))
		} else {
			for _, entry := range oceanData.Properties.Timeseries {
				parsedTime, err := time.Parse(time.RFC3339, entry.Time)
				if err != nil {
					log.Printf("Skipping ocean forecast due to invalid time format: %v", err)
					continue
				}
				ts := parsedTime.Unix()
				forecast := ensureForecast(forecasts, ts)
				st := entry.Data.Instant.Details.SeaWaterTemperature
				forecast.SeaTemperature = &st
				wh := entry.Data.Instant.Details.SeaSurfaceWaveHeight
				forecast.WaveHeight = &wh
				wd := entry.Data.Instant.Details.SeaSurfaceWaveFromDirection
				forecast.WaveDirection = &wd
			}
		}
	}

	if weatherData != nil {
		if len(weatherData.Geometry.Coordinates) >= 2 {
			response.ForecastPosition = &Coordinates{weatherData.Geometry.Coordinates[1], weatherData.Geometry.Coordinates[0]}
		}

		if weatherData.Properties.Meta.Error != nil {
			appendResponseError(&response, fmt.Sprintf("weather API error: %v", weatherData.Properties.Meta.Error))
		} else {
			for _, entry := range weatherData.Properties.Timeseries {
				parsedTime, err := time.Parse(time.RFC3339, entry.Time)
				if err != nil {
					log.Printf("Skipping weather forecast due to invalid time format: %v", err)
					continue
				}
				ts := parsedTime.Unix()
				forecast := ensureForecast(forecasts, ts)
				forecast.Temperature = floatPtr(entry.Data.Instant.Details.AirTemperature)
				forecast.WindSpeed = floatPtr(entry.Data.Instant.Details.WindSpeed)
				forecast.WindDirection = floatPtr(entry.Data.Instant.Details.WindFromDirection)
				forecast.CloudCover = buildCloudCover(entry)
				forecast.UvIndex = floatPtr(entry.Data.Instant.Details.UltravioletIndexClearSky)

				condition := mapSymbolToCondition(entry.Data.Next1Hours.Summary.SymbolCode, entry)
				if condition != "" {
					forecast.Condition = &condition
				}

				if entry.Data.Next12Hours.Details.ProbabilityOfPrecipitation != nil {
					forecast.Precipitation = entry.Data.Next12Hours.Details.ProbabilityOfPrecipitation
				} else {
					fallback := 0.0
					if symbolImpliesPrecipitation(entry.Data.Next12Hours.Summary.SymbolCode) {
						fallback = 100.0
					}
					forecast.Precipitation = &fallback
				}
			}
		}
	}

	forecastSlice := make([]Forecast, 0, len(forecasts))
	for _, f := range forecasts {
		forecastSlice = append(forecastSlice, *f)
	}

	sort.Slice(forecastSlice, func(i, j int) bool {
		return forecastSlice[i].Time < forecastSlice[j].Time
	})

	if len(forecastSlice) > forecastEntryLimit {
		forecastSlice = forecastSlice[:forecastEntryLimit]
	}

	response.Forecast = forecastSlice

	if response.Error == nil && len(response.Forecast) == 0 {
		response.Error = "No timeseries data available from any source"
	}

	return response
}

func buildCompactResponse(response ApiResponseJSON) ApiResponseCompact {
	compact := ApiResponseCompact{
		RequestPosition:       response.RequestPosition,
		ForecastPosition:      response.ForecastPosition,
		OceanForecastPosition: response.OceanForecastPosition,
		RequestTime:           response.RequestTime,
		Error:                 response.Error,
	}

	if len(response.Forecast) == 0 {
		return compact
	}

	compact.Forecast = make([][]any, len(response.Forecast))
	for i, forecast := range response.Forecast {
		compact.Forecast[i] = forecastToArray(forecast)
	}

	return compact
}

func ensureForecast(cache map[int64]*Forecast, ts int64) *Forecast {
	if existing, ok := cache[ts]; ok {
		return existing
	}
	cache[ts] = &Forecast{Time: ts}
	return cache[ts]
}

func appendResponseError(response *ApiResponseJSON, extra string) {
	if response.Error == nil {
		response.Error = extra
		return
	}
	response.Error = fmt.Sprintf("%v; %s", response.Error, extra)
}

func floatPtr(value float64) *float64 {
	return &value
}

func buildCloudCover(entry WeatherTimeseriesEntry) []float64 {
	cloudCover := []float64{entry.Data.Instant.Details.CloudAreaFraction}
	if entry.Data.Instant.Details.CloudAreaFractionLow != nil &&
		entry.Data.Instant.Details.CloudAreaFractionMedium != nil &&
		entry.Data.Instant.Details.CloudAreaFractionHigh != nil {
		cloudCover = append(cloudCover,
			*entry.Data.Instant.Details.CloudAreaFractionLow,
			*entry.Data.Instant.Details.CloudAreaFractionMedium,
			*entry.Data.Instant.Details.CloudAreaFractionHigh,
		)
	}
	return cloudCover
}

func mapSymbolToCondition(symbolCode string, entry WeatherTimeseriesEntry) string {
	if symbolCode == "" {
		return ""
	}

	lowerSymbol := strings.ToLower(symbolCode)
	if strings.Contains(lowerSymbol, "thunder") || strings.Contains(lowerSymbol, "storm") {
		return "thunder"
	}
	if strings.Contains(lowerSymbol, "hail") {
		return "hail"
	}
	if strings.Contains(lowerSymbol, "fog") {
		return "fog"
	}
	if strings.Contains(lowerSymbol, "snow") || strings.Contains(lowerSymbol, "sleet") {
		return "snow"
	}
	if strings.Contains(lowerSymbol, "rain") {
		if strings.Contains(lowerSymbol, "light") {
			return "light rain"
		}
		return "rain"
	}

	symbol := strings.Split(lowerSymbol, "_")[0]
	switch symbol {
	case "clearsky", "fair", "partlycloudy", "cloudy":
		cloudCondition := interpretClouds(
			entry.Data.Instant.Details.CloudAreaFraction,
			entry.Data.Instant.Details.CloudAreaFractionLow,
			entry.Data.Instant.Details.CloudAreaFractionMedium,
		)
		if cloudCondition != "" {
			return cloudCondition
		}
		switch symbol {
		case "clearsky":
			return "clear"
		case "fair":
			return "fair"
		case "partlycloudy":
			return "partly cloudy"
		default:
			return "cloudy"
		}
	}

	return ""
}

func interpretClouds(total float64, low, medium *float64) string {
	total = clampCloudCover(total)
	if total >= 90 {
		if low != nil && *low >= 70 {
			return "cloudy low"
		}
		if low != nil && medium != nil && *low < 50 && *medium < 50 {
			return "cloudy high"
		}
		return "cloudy"
	}
	if total >= 70 {
		return "partly cloudy 80"
	}
	if total >= 50 {
		return "partly cloudy 60"
	}
	if total >= 30 {
		return "partly cloudy 40"
	}
	if total >= 10 {
		return "partly cloudy 20"
	}
	return ""
}

func clampCloudCover(value float64) float64 {
	return math.Max(0, math.Min(100, value))
}

func forecastToArray(f Forecast) []any {
	entry := make([]any, ForecastEntrySize)
	entry[ForecastIdxTime] = f.Time

	if f.SeaTemperature != nil {
		entry[ForecastIdxSeaTemperature] = *f.SeaTemperature
	}
	if f.WaveHeight != nil {
		entry[ForecastIdxWaveHeight] = *f.WaveHeight
	}
	if f.WaveDirection != nil {
		entry[ForecastIdxWaveDirection] = *f.WaveDirection
	}
	if f.Temperature != nil {
		entry[ForecastIdxTemperature] = *f.Temperature
	}
	if f.WindSpeed != nil {
		entry[ForecastIdxWindSpeed] = *f.WindSpeed
	}
	if f.WindDirection != nil {
		entry[ForecastIdxWindDirection] = *f.WindDirection
	}
	if len(f.CloudCover) > 0 {
		entry[ForecastIdxCloudCover] = f.CloudCover
	}
	if f.Condition != nil {
		entry[ForecastIdxCondition] = *f.Condition
	}
	if f.UvIndex != nil {
		entry[ForecastIdxUvIndex] = *f.UvIndex
	}
	if f.Precipitation != nil {
		entry[ForecastIdxPrecipitation] = *f.Precipitation
	}

	return entry
}

func symbolImpliesPrecipitation(symbolCode string) bool {
	if symbolCode == "" {
		return false
	}

	lowerSymbol := strings.ToLower(symbolCode)
	keywords := []string{"rain", "sleet", "snow", "shower", "thunder", "hail"}
	for _, kw := range keywords {
		if strings.Contains(lowerSymbol, kw) {
			return true
		}
	}
	return false
}
