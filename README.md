# Wake Service

Wake is a small Go service that proxies the YR ocean and location forecasts, adds caching, and exposes a compact or JSON-readable format for clients like Garmin watch faces and AI agents. It also supports logging watch locations and scheduling OpenClaw prompts after a workout.

Data is provided by the [Norwegian Meteorological Institute (YR)](https://api.met.no).

## Endpoints

### `GET /weather`

Query parameters:

- `lat` (float, required).
- `lon` (float, required).
- `format` (optional): `json` (default) or `compact`.
- `logLocation` (optional): `true`/`1` to log the request position (requires an API key with location logging enabled).

The response includes merged ocean + weather forecast data. Times without location forecast data (ocean-only samples) are omitted. The `compact` format uses arrays for each forecast entry (limited to 12 entries and omits cloud cover). The `json` format uses objects with named keys, including a `cloudCover` object with `total`, `low`, `medium`, and `high` fields when available; it returns all available forecast times. JSON timestamps are formatted in local time, precipitation is reported as `precipitation12hours`, and `meta.units` describes the units.

Weather conditions are interpreted server-side, including cloud overlays and heavy/high cloud variants. The `condition` values are:

- `clear`
- `fair`
- `partly cloudy 20`
- `partly cloudy 40`
- `partly cloudy 60`
- `partly cloudy 80`
- `cloudy`
- `heavy clouds`
- `high clouds`
- `light rain`
- `rain`
- `thunder`
- `snow`
- `hail`
- `fog`

### `GET /sync`

Returns the same weather response as `/weather` and optionally schedules OpenClaw when the workout marker changes.

Query parameters:

- `lat` (float, required).
- `lon` (float, required).
- `format` (optional): `json` (default) or `compact`.
- `logLocation` (optional): `true`/`1` to log the request position (requires an API key with location logging enabled).
- `lastWorkout` (int, required): timestamp-like marker for the most recent workout. The value is treated as an opaque number and only compared to the last stored value. `0` is treated as "no workout" and is ignored.

When `lastWorkout` differs from the stored value (and is non-zero), the service schedules the OpenClaw prompt (requires an API key with workout permissions).

## Options file

The service reads configuration from a JSON file. Set the path with `WAKE_OPTIONS_PATH`. Defaults:

- Linux: `/etc/wake-service/options.json`
- macOS: `~/Library/Application Support/wake-service/options.json`

Example:

```json
{
  "userAgent": "Wake/1.0 (you@example.com)",
  "locationLogPath": "/var/wake-service/location.json",
  "syncStatePath": "/var/wake-service/sync-state.json",
  "apiKeys": [
    {
      "name": "watch",
      "key": "change-me",
      "allowLocationLog": true,
      "allowWorkout": true
    }
  ],
  "openClaw": {
    "delayMinutes": 3,
    "prompt": "The user has logged an activity with Garmin. Check Garmin stats and give feedback."
  }
}
```

Notes:

- `userAgent` is required and must be a descriptive identifier for the MET API.
- Location logging only happens for API keys with `allowLocationLog`.
- Workout scheduling only works for API keys with `allowWorkout`.
- If `delayMinutes` is omitted, it defaults to 3 minutes; `0` is allowed to run immediately.
- The workout prompt is sent to OpenClaw as-is.

## Build & Run

```bash
go build ./...
PORT=8080 ./wake-service
```

## Deployment
Deploy your own instance of this service to use it (I don't want your location data). Easily deployed to any server - your favourite AI agent can help you with this!

## AI usage
All code prompted with Codex & reviewed.
