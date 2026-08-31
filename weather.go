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
	compactForecastLimit = 12
)

const (
	ForecastIdxTime = iota
	ForecastIdxSeaTemperature
	ForecastIdxWaveHeight
	ForecastIdxWaveDirection
	ForecastIdxTemperature
	ForecastIdxWindSpeed
	ForecastIdxWindDirection
	ForecastIdxCondition
	ForecastIdxUvIndex
	ForecastIdxPrecipitation
	ForecastIdxCloudCover
	ForecastEntrySize
)

func buildForecastResponse(oceanData *OceanForecastData, weatherData *WeatherYrResponse, nowcastData *NowcastYrResponse, requestPos Position, errors []string) ApiResponseJSON {
	requestTime := time.Now()
	response := ApiResponseJSON{
		Meta: &ResponseMeta{
			Units: ForecastUnits{
				Time:                     "local",
				SeaTemperature:           "celsius",
				WaveHeight:               "meters",
				WaveDirection:            "degrees",
				Temperature:              "celsius",
				WindSpeed:                "m/s",
				WindDirection:            "degrees",
				CloudCover:               "percent",
				Condition:                "text",
				UvIndex:                  "index",
				Precipitation1Hour:       "percent",
				PrecipitationAmount1Hour: "mm",
				Precipitation12Hours:     "percent",
			},
		},
		RequestPosition: Coordinates{requestPos.Lat, requestPos.Lon},
		RequestTime:     requestTime.Unix(),
	}

	if len(errors) > 0 {
		response.Error = strings.Join(errors, "; ")
	}

	forecasts := make(map[int64]*Forecast)
	weatherTimes := make(map[int64]bool)

	if oceanData != nil {
		response.OceanForecastPosition = oceanData.Position

		if oceanData.APIError != nil {
			appendResponseError(&response, fmt.Sprintf("ocean API error: %v", oceanData.APIError))
		} else {
			for _, entry := range oceanData.Timeseries {
				forecast := ensureForecast(forecasts, entry.Time)
				forecast.SeaTemperature = entry.SeaTemperature
				forecast.WaveHeight = entry.WaveHeight
				forecast.WaveDirection = entry.WaveDirection
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
				weatherTimes[ts] = true
				forecast := ensureForecast(forecasts, ts)
				forecast.Temperature = floatPtr(entry.Data.Instant.Details.AirTemperature)
				forecast.WindSpeed = floatPtr(entry.Data.Instant.Details.WindSpeed)
				forecast.WindDirection = floatPtr(entry.Data.Instant.Details.WindFromDirection)
				forecast.CloudCover = buildCloudCover(entry)
				forecast.UvIndex = floatPtr(entry.Data.Instant.Details.UltravioletIndexClearSky)
				forecast.Precipitation1Hour = entry.Data.Next1Hours.Details.ProbabilityOfPrecipitation
				forecast.PrecipitationAmount1Hour = entry.Data.Next1Hours.Details.PrecipitationAmount

				condition := mapSymbolToCondition(entry.Data.Next1Hours.Summary.SymbolCode, entry)
				if condition != "" {
					forecast.Condition = &condition
				}

				if entry.Data.Next12Hours.Details.ProbabilityOfPrecipitation != nil {
					forecast.Precipitation12Hours = entry.Data.Next12Hours.Details.ProbabilityOfPrecipitation
				} else {
					fallback := 0.0
					if symbolImpliesPrecipitation(entry.Data.Next12Hours.Summary.SymbolCode) {
						fallback = 100.0
					}
					forecast.Precipitation12Hours = &fallback
				}
			}
		}
	}

	applyNowcast(forecasts, weatherTimes, nowcastData, requestTime)

	forecastSlice := make([]Forecast, 0, len(forecasts))
	for _, f := range forecasts {
		if len(weatherTimes) > 0 {
			if !weatherTimes[f.TimeUnix] {
				continue
			}
		}
		forecastSlice = append(forecastSlice, *f)
	}

	sort.Slice(forecastSlice, func(i, j int) bool {
		return forecastSlice[i].TimeUnix < forecastSlice[j].TimeUnix
	})

	response.Forecast = forecastSlice

	if response.Error == nil && len(response.Forecast) == 0 {
		response.Error = "No timeseries data available from any source"
	}

	return response
}

func applyNowcast(forecasts map[int64]*Forecast, weatherTimes map[int64]bool, data *NowcastYrResponse, now time.Time) {
	if data == nil || data.Properties.Meta.RadarCoverage != "ok" || len(data.Properties.Timeseries) == 0 {
		return
	}

	currentHour := now.Truncate(time.Hour).Unix()
	if !weatherTimes[currentHour] {
		return
	}

	entry := data.Properties.Timeseries[0]
	entryTime, err := time.Parse(time.RFC3339, entry.Time)
	if err != nil {
		log.Printf("Skipping nowcast due to invalid time format: %v", err)
		return
	}
	if delta := entryTime.Sub(now); delta < -10*time.Minute || delta > 10*time.Minute {
		return
	}

	// Nowcast rounds its first point to a five-minute boundary, which can fall
	// just inside the next hour. It still represents current conditions, so apply
	// it to Locationforecast's current hourly entry.
	forecast := forecasts[currentHour]
	details := entry.Data.Instant.Details
	if details.AirTemperature != nil {
		forecast.Temperature = details.AirTemperature
	}
	if details.WindSpeed != nil {
		forecast.WindSpeed = details.WindSpeed
	}
	if details.WindFromDirection != nil {
		forecast.WindDirection = details.WindFromDirection
	}
	if details.UltravioletIndexClearSky != nil {
		forecast.UvIndex = details.UltravioletIndexClearSky
	}
	if entry.Data.Next1Hours.Details.PrecipitationAmount != nil {
		forecast.PrecipitationAmount1Hour = entry.Data.Next1Hours.Details.PrecipitationAmount
	}
	if nowcastIndicatesPrecipitation(entry) {
		certainty := 100.0
		forecast.Precipitation12Hours = &certainty

		condition := mapNowcastSymbolToCondition(entry.Data.Next1Hours.Summary.SymbolCode)
		if condition != "" {
			forecast.Condition = &condition
		}
	}
}

func nowcastIndicatesPrecipitation(entry NowcastTimeseriesEntry) bool {
	if rate := entry.Data.Instant.Details.PrecipitationRate; rate != nil && *rate > 0 {
		return true
	}
	if amount := entry.Data.Next1Hours.Details.PrecipitationAmount; amount != nil && *amount > 0 {
		return true
	}
	return symbolImpliesPrecipitation(entry.Data.Next1Hours.Summary.SymbolCode)
}

func mapNowcastSymbolToCondition(symbolCode string) string {
	entry := WeatherTimeseriesEntry{}
	return mapSymbolToCondition(symbolCode, entry)
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

	forecastCount := len(response.Forecast)
	if forecastCount > compactForecastLimit {
		forecastCount = compactForecastLimit
	}

	compact.Forecast = make([][]any, forecastCount)
	for i := 0; i < forecastCount; i++ {
		compact.Forecast[i] = forecastToCompactArray(response.Forecast[i])
	}

	return compact
}

func ensureForecast(cache map[int64]*Forecast, ts int64) *Forecast {
	if existing, ok := cache[ts]; ok {
		return existing
	}
	cache[ts] = &Forecast{TimeUnix: ts, Time: formatForecastTime(ts)}
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

func formatForecastTime(ts int64) string {
	return time.Unix(ts, 0).In(time.Local).Format("2006-01-02 15:04:05")
}

func buildCloudCover(entry WeatherTimeseriesEntry) *CloudCover {
	cover := &CloudCover{Total: entry.Data.Instant.Details.CloudAreaFraction}
	cover.Low = entry.Data.Instant.Details.CloudAreaFractionLow
	cover.Medium = entry.Data.Instant.Details.CloudAreaFractionMedium
	cover.High = entry.Data.Instant.Details.CloudAreaFractionHigh
	return cover
}

func mapSymbolToCondition(symbolCode string, entry WeatherTimeseriesEntry) string {
	if symbolCode == "" {
		return ""
	}

	lowerSymbol := strings.ToLower(symbolCode)
	if strings.Contains(lowerSymbol, "thunder") {
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
			return "heavy clouds"
		}
		if low != nil && medium != nil && *low < 50 && *medium < 50 {
			return "high clouds"
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
	entry[ForecastIdxTime] = f.TimeUnix

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
	if f.Condition != nil {
		entry[ForecastIdxCondition] = *f.Condition
	}
	if f.UvIndex != nil {
		entry[ForecastIdxUvIndex] = *f.UvIndex
	}
	if f.Precipitation12Hours != nil {
		entry[ForecastIdxPrecipitation] = *f.Precipitation12Hours
	}
	if f.CloudCover != nil {
		entry[ForecastIdxCloudCover] = cloudCoverToArray(f.CloudCover)
	}

	return entry
}

func forecastToCompactArray(f Forecast) []any {
	entry := make([]any, ForecastEntrySize-1)
	entry[ForecastIdxTime] = f.TimeUnix

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
	if f.Condition != nil {
		entry[ForecastIdxCondition] = *f.Condition
	}
	if f.UvIndex != nil {
		entry[ForecastIdxUvIndex] = *f.UvIndex
	}
	if f.Precipitation12Hours != nil {
		entry[ForecastIdxPrecipitation] = *f.Precipitation12Hours
	}

	return entry
}

func cloudCoverToArray(cloudCover *CloudCover) []float64 {
	values := []float64{cloudCover.Total}
	if cloudCover.Low != nil && cloudCover.Medium != nil && cloudCover.High != nil {
		values = append(values, *cloudCover.Low, *cloudCover.Medium, *cloudCover.High)
	}
	return values
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
