package main

import "time"

type Position struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Coordinates [2]float64

type Forecast struct {
	Time                      string      `json:"time"`
	TimeUnix                  int64       `json:"-"`
	SeaTemperature            *float64    `json:"seaTemperature,omitempty"`
	WaveHeight                *float64    `json:"waveHeight,omitempty"`
	WaveDirection             *float64    `json:"waveDirection,omitempty"`
	Temperature               *float64    `json:"temperature,omitempty"`
	WindSpeed                 *float64    `json:"windSpeed,omitempty"`
	WindDirection             *float64    `json:"windDirection,omitempty"`
	CloudCover                *CloudCover `json:"cloudCover,omitempty"`
	Condition                 *string     `json:"condition,omitempty"`
	UvIndex                   *float64    `json:"uvIndex,omitempty"`
	Precipitation1Hours       *float64    `json:"precipitation1hours,omitempty"`
	PrecipitationAmount1Hours *float64    `json:"precipitationAmount1hours,omitempty"`
	Precipitation12Hours      *float64    `json:"precipitation12hours,omitempty"`
}

type CloudCover struct {
	Total  float64  `json:"total"`
	Low    *float64 `json:"low,omitempty"`
	Medium *float64 `json:"medium,omitempty"`
	High   *float64 `json:"high,omitempty"`
}

type ApiResponseJSON struct {
	Meta                  *ResponseMeta `json:"meta,omitempty"`
	RequestPosition       Coordinates   `json:"requestPosition"`
	ForecastPosition      *Coordinates  `json:"forecastPosition,omitempty"`
	OceanForecastPosition *Coordinates  `json:"oceanForecastPosition,omitempty"`
	RequestTime           int64         `json:"requestTime"`
	Forecast              []Forecast    `json:"forecast,omitempty"`
	Error                 interface{}   `json:"error,omitempty"`
}

type ResponseMeta struct {
	Units ForecastUnits `json:"units"`
}

type ForecastUnits struct {
	Time                      string `json:"time"`
	SeaTemperature            string `json:"seaTemperature"`
	WaveHeight                string `json:"waveHeight"`
	WaveDirection             string `json:"waveDirection"`
	Temperature               string `json:"temperature"`
	WindSpeed                 string `json:"windSpeed"`
	WindDirection             string `json:"windDirection"`
	CloudCover                string `json:"cloudCover"`
	Condition                 string `json:"condition"`
	UvIndex                   string `json:"uvIndex"`
	Precipitation1Hours       string `json:"precipitation1hours"`
	PrecipitationAmount1Hours string `json:"precipitationAmount1hours"`
	Precipitation12Hours      string `json:"precipitation12hours"`
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
