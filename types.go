package main

import "time"

type Position struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Coordinates [2]float64

type Forecast struct {
	Time           int64     `json:"time"`
	SeaTemperature *float64  `json:"seaTemperature,omitempty"`
	WaveHeight     *float64  `json:"waveHeight,omitempty"`
	WaveDirection  *float64  `json:"waveDirection,omitempty"`
	Temperature    *float64  `json:"temperature,omitempty"`
	WindSpeed      *float64  `json:"windSpeed,omitempty"`
	WindDirection  *float64  `json:"windDirection,omitempty"`
	CloudCover     []float64 `json:"cloudCover,omitempty"`
	Condition      *string   `json:"condition,omitempty"`
	UvIndex        *float64  `json:"uvIndex,omitempty"`
	Precipitation  *float64  `json:"precipitation,omitempty"`
}

type ApiResponseJSON struct {
	RequestPosition       Coordinates  `json:"requestPosition"`
	ForecastPosition      *Coordinates `json:"forecastPosition,omitempty"`
	OceanForecastPosition *Coordinates `json:"oceanForecastPosition,omitempty"`
	RequestTime           int64        `json:"requestTime"`
	Forecast              []Forecast   `json:"forecast,omitempty"`
	Error                 interface{}  `json:"error,omitempty"`
}

type ApiResponseCompact struct {
	RequestPosition       Coordinates  `json:"requestPosition"`
	ForecastPosition      *Coordinates `json:"forecastPosition,omitempty"`
	OceanForecastPosition *Coordinates `json:"oceanForecastPosition,omitempty"`
	RequestTime           int64        `json:"requestTime"`
	Forecast              [][]any      `json:"forecast,omitempty"`
	Error                 interface{}  `json:"error,omitempty"`
}

type CacheEntry struct {
	Created time.Time
	Data    ApiResponseJSON
}
