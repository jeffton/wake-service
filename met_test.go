package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestFetchOceanDataUsesOpenMeteoOutsideYrCoverage(t *testing.T) {
	var openMeteoCalls int
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "api.met.no":
			response := testHTTPResponse(http.StatusUnprocessableEntity, "outside coverage")
			response.Header.Set("X-ErrorClass", "Outsidegeographicarea")
			return response, nil
		case "marine-api.open-meteo.com":
			openMeteoCalls++
			query := req.URL.Query()
			if got := query.Get("latitude"); got != "37.7540" {
				t.Errorf("latitude = %q, want 37.7540", got)
			}
			if got := query.Get("longitude"); got != "26.9780" {
				t.Errorf("longitude = %q, want 26.9780", got)
			}
			if got := query.Get("hourly"); got != "wave_height,wave_direction,sea_surface_temperature" {
				t.Errorf("hourly = %q", got)
			}
			if got := query.Get("timeformat"); got != "unixtime" {
				t.Errorf("timeformat = %q, want unixtime", got)
			}
			if got := query.Get("cell_selection"); got != "sea" {
				t.Errorf("cell_selection = %q, want sea", got)
			}
			return testHTTPResponse(http.StatusOK, `{
				"latitude":37.791664,
				"longitude":26.958344,
				"hourly":{
					"time":[1785801600,1785805200],
					"wave_height":[0.42,null],
					"wave_direction":[315,316],
					"sea_surface_temperature":[24.8,24.9]
				}
			}`), nil
		default:
			t.Fatalf("unexpected request host: %s", req.URL.Host)
			return nil, nil
		}
	})}

	data, rawBody, err := fetchOceanData(client, "Wake test", Position{Lat: 37.754, Lon: 26.978})
	if err != nil {
		t.Fatalf("fetch ocean data: %v (body %s)", err, rawBody)
	}
	if openMeteoCalls != 1 {
		t.Fatalf("Open-Meteo calls = %d, want 1", openMeteoCalls)
	}
	if data.Position == nil || *data.Position != (Coordinates{37.791664, 26.958344}) {
		t.Fatalf("position = %v", data.Position)
	}
	if len(data.Timeseries) != 2 {
		t.Fatalf("timeseries length = %d, want 2", len(data.Timeseries))
	}
	first := data.Timeseries[0]
	if first.Time != 1785801600 || first.SeaTemperature == nil || *first.SeaTemperature != 24.8 || first.WaveHeight == nil || *first.WaveHeight != 0.42 || first.WaveDirection == nil || *first.WaveDirection != 315 {
		t.Fatalf("unexpected first entry: %+v", first)
	}
	if data.Timeseries[1].WaveHeight != nil {
		t.Fatalf("null wave height = %v, want nil", data.Timeseries[1].WaveHeight)
	}
}

func TestFetchOceanDataDoesNotFallbackForOtherYrErrors(t *testing.T) {
	var openMeteoCalls int
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "marine-api.open-meteo.com" {
			openMeteoCalls++
		}
		return testHTTPResponse(http.StatusInternalServerError, "upstream failure"), nil
	})}

	_, rawBody, err := fetchOceanData(client, "Wake test", Position{})
	if err == nil {
		t.Fatal("expected Yr error")
	}
	httpErr, ok := err.(*HTTPStatusError)
	if !ok || httpErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("error = %T %v, want HTTPStatusError 500", err, err)
	}
	if string(rawBody) != "upstream failure" {
		t.Fatalf("body = %q", rawBody)
	}
	if openMeteoCalls != 0 {
		t.Fatalf("Open-Meteo calls = %d, want 0", openMeteoCalls)
	}
}

func TestFetchOceanDataDoesNotFallbackForUnrelatedYr422(t *testing.T) {
	var openMeteoCalls int
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "marine-api.open-meteo.com" {
			openMeteoCalls++
		}
		response := testHTTPResponse(http.StatusUnprocessableEntity, "invalid parameters")
		response.Header.Set("X-ErrorClass", "InvalidParameter")
		return response, nil
	})}

	_, _, err := fetchOceanData(client, "Wake test", Position{})
	if err == nil {
		t.Fatal("expected Yr error")
	}
	if openMeteoCalls != 0 {
		t.Fatalf("Open-Meteo calls = %d, want 0", openMeteoCalls)
	}
}

func TestOpenMeteoRejectsResponseWithoutMarineData(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusOK, `{
			"latitude":40,
			"longitude":20,
			"hourly":{
				"time":[1785801600],
				"wave_height":[null],
				"wave_direction":[null],
				"sea_surface_temperature":[null]
			}
		}`), nil
	})}

	_, _, err := fetchOpenMeteoMarineData(client, "Wake test", Position{})
	if err == nil || !strings.Contains(err.Error(), "no marine forecast data") {
		t.Fatalf("error = %v, want no marine forecast data", err)
	}
}

func TestFetchOceanDataKeepsYrInsideCoverage(t *testing.T) {
	var openMeteoCalls int
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "marine-api.open-meteo.com" {
			openMeteoCalls++
		}
		return testHTTPResponse(http.StatusOK, `{
			"geometry":{"coordinates":[10,59]},
			"properties":{"timeseries":[{
				"time":"2026-08-04T12:00:00Z",
				"data":{"instant":{"details":{
					"sea_water_temperature":18.2,
					"sea_surface_wave_height":0.7,
					"sea_surface_wave_from_direction":220
				}}}
			}]}
		}`), nil
	})}

	data, _, err := fetchOceanData(client, "Wake test", Position{Lat: 59, Lon: 10})
	if err != nil {
		t.Fatalf("fetch ocean data: %v", err)
	}
	if openMeteoCalls != 0 {
		t.Fatalf("Open-Meteo calls = %d, want 0", openMeteoCalls)
	}
	if len(data.Timeseries) != 1 || data.Timeseries[0].SeaTemperature == nil || *data.Timeseries[0].SeaTemperature != 18.2 {
		t.Fatalf("unexpected Yr data: %+v", data)
	}
}
