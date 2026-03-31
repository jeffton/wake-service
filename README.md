# Wake Service

Wake is a small Go service that proxies the YR ocean and location forecasts, adds caching, and exposes a compact or JSON-readable format for clients like Garmin watch faces and AI agents. It also supports logging watch locations and scheduling a configurable cron command after a workout.

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

Returns the same weather response as `/weather` and optionally schedules the configured cron command when the workout marker changes.

Query parameters:

- `lat` (float, required).
- `lon` (float, required).
- `format` (optional): `json` (default) or `compact`.
- `logLocation` (optional): `true`/`1` to log the request position (requires an API key with location logging enabled).
- `lastWorkout` (int, required): timestamp-like marker for the most recent workout. The value is treated as an opaque number and only compared to the last stored value. `0` is treated as "no workout" and is ignored.

When `lastWorkout` differs from the stored value (and is non-zero), the service runs the configured cron command (requires an API key with workout permissions).

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
  "cron": {
    "command": "openclaw cron add --name \"Garmin workout ping\" --delete-after-run --system-event {prompt} --at \"3m\"",
    "prompt": "David har logget en ny Garmin-aktivitet i /root/github/pt. Følg instruktionerne i AGENTS.md."
  }
}
```

Notes:

- `userAgent` is required and must be a descriptive identifier for the MET API.
- Location logging only happens for API keys with `allowLocationLog`.
- Workout scheduling only works for API keys with `allowWorkout`.
- `cron.command` is required.
- `cron.prompt` is required.
- `cron.command` is executed through `/bin/sh -c`.
- Wake replaces `{prompt}` in `cron.command` with the configured prompt, shell-escaped as a single argument.
- Any scheduling delay should be encoded directly in `cron.command`.
- To target Batty instead of OpenClaw, use a Batty CLI command such as `batty --root /root/github cron add --workspace pt --prompt {prompt} --in "3m"`.

## Build & Run

```bash
go build ./...
PORT=8080 ./wake-service
```

## Deployment
Deploy your own instance of this service to use it (I don't want your location data). Easily deployed to any server - your favourite AI agent can help you with this!

## AI usage
All code prompted with Codex & reviewed.
