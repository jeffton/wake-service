# Wake Service

Wake is a small Go service that proxies the MET Norway (YR) ocean and location forecasts, adds caching, and exposes a compact or JSON-readable format for clients like Garmin watch faces and AI agents. It also supports logging watch locations and scheduling OpenClaw prompts after a workout.

Data is provided by the Norwegian Meteorological Institute (YR), https://api.met.no.

## Endpoints

### `GET /weather`

Query parameters:

- `lat` (float, required).
- `lon` (float, required).
- `format` (optional): `json` (default) or `compact`.
- `logLocation` (optional): `true`/`1` to log the request position (requires an API key with location logging enabled).

The response includes merged ocean + weather forecast data. The `compact` format uses arrays for each forecast entry. The `json` format uses objects with named keys.

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
- `storm`
- `thunder`
- `snow`
- `hail`
- `fog`

### `POST /workout`

Schedules an OpenClaw cron job for activity feedback.

Request body:

```json
{
  "activityCount": 2
}
```

Response (success):

```json
{ "ok": true }
```

Response (error):

```json
{ "ok": false, "error": "..." }
```

## Options file

The service reads configuration from a JSON file. Set the path with `WAKE_OPTIONS_PATH`. Defaults:

- Linux: `/etc/wake-service/options.json`
- macOS: `~/Library/Application Support/wake-service/options.json`

Example:

```json
{
  "userAgent": "Wake/1.0 (you@example.com)",
  "locationLogPath": "/var/log/wake/location.json",
  "apiKeys": [
    {
      "name": "watch",
      "key": "change-me",
      "allowLocationLog": true,
      "allowWorkout": true
    }
  ],
  "openClaw": {
    "url": "http://localhost:18789",
    "token": "replace-with-token",
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
