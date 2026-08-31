# Wake Service

Wake is a small Go service that proxies YR location, precipitation nowcast, and ocean forecasts, adds caching, stores a trusted location, and exposes compact or JSON-readable weather responses for clients like Garmin watch faces and AI agents. It can also schedule a configurable cron command after a workout. Outside the area covered by YR Oceanforecast, marine data is fetched from Open-Meteo.

Weather and regional ocean data are provided by the [Norwegian Meteorological Institute (YR)](https://api.met.no). The global marine fallback is provided by [Open-Meteo](https://open-meteo.com/en/docs/marine-weather-api), using marine models from providers including [DWD](https://www.dwd.de/), Météo-France/Copernicus, ECMWF, and NOAA.

## Endpoints

All endpoints require an `X-Api-Key` header.

### `GET /location`

Returns the stored location. Requires a `full` API key.

Response:

```json
{
  "lat": 59.1234,
  "lon": 10.1234,
  "precision": "reduced",
  "ts": 1779020000
}
```

### `POST /location`

Stores the location. Requires a `full` API key. All parameters are supplied in the JSON body.

JSON body:

```json
{
  "lat": 59.1234,
  "lon": 10.1234,
  "precision": "reduced"
}

`precision` is optional and stored as supplied by the client.
```

### `GET /weather`

Fetches weather for supplied coordinates, or for the stored location when coordinates are omitted. Using the stored location requires a `full` API key. A `weather` API key may fetch weather only for supplied coordinates.

Query parameters:

- `lat` (float, optional).
- `lon` (float, optional).
- `format` (optional): `json` (default) or `compact`.

The response includes merged ocean + weather forecast data. Within MET's Nordic radar coverage, the current hourly entry uses precipitation amount, temperature, and wind from the radar-based Nowcast API. Its condition uses Nowcast when precipitation is detected and Locationforecast otherwise; later entries use Locationforecast. Weather responses are cached for two minutes to stay close to the Nowcast update cadence. Open-Meteo is used for sea-surface temperature, wave height, and wave direction only when YR reports that the coordinates are outside its Oceanforecast coverage. Some of its marine datasets use an approximately 8 km grid; resolution varies by model. The fallback is suitable for weather reports, not coastal navigation. Times without location forecast data (ocean-only samples) are omitted. The `compact` format uses arrays for each forecast entry (limited to 12 entries and omits cloud cover). The `json` format uses objects with named keys, including a `cloudCover` object with `total`, `low`, `medium`, and `high` fields when available; it returns all available forecast times. JSON timestamps are formatted in local time. Precipitation probability is reported as `precipitation1hour` and `precipitation12hours`, and the next-hour amount is reported as `precipitationAmount1hour` when available. Radar-confirmed or Nowcast-predicted precipitation within the next hour makes `precipitation12hours` 100%; otherwise its Locationforecast value is preserved. `meta.units` describes the units.

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

### `POST /calendar`

Replaces the stored calendar snapshot. Requires a `full` API key. Batty Companion sends selected calendars and upcoming events for the next 7 days.

JSON body:

```json
{
  "syncedAt": "2026-06-02T20:00:00Z",
  "windowStart": "2026-06-02T20:00:00Z",
  "windowEnd": "2026-06-09T20:00:00Z",
  "calendars": [
    {
      "id": "device-calendar-id",
      "title": "Personal",
      "sourceTitle": "Google",
      "sourceType": "calDAV",
      "colorHex": "#34A853"
    }
  ],
  "events": [
    {
      "id": "event-id",
      "eventIdentifier": "event-id",
      "calendarItemIdentifier": "calendar-item-id",
      "calendarItemExternalIdentifier": "google-event-id",
      "calendarIdentifier": "device-calendar-id",
      "calendarTitle": "Personal",
      "sourceTitle": "Google",
      "title": "Dentist",
      "location": "Clinic",
      "notes": "Bring card",
      "url": "https://example.com/event",
      "startDate": "2026-06-03T09:00:00Z",
      "endDate": "2026-06-03T10:00:00Z",
      "timeZone": "Europe/Oslo",
      "isAllDay": false,
      "availability": "busy",
      "status": "confirmed",
      "organizer": null,
      "attendees": [],
      "alarms": [],
      "recurrenceRules": [],
      "lastModifiedDate": "2026-06-01T12:00:00Z",
      "creationDate": "2026-05-20T12:00:00Z"
    }
  ]
}
```

The service writes the posted snapshot to `calendar.json` in `dataDir` atomically. It replaces the previous snapshot instead of merging. Event identifiers, calendar identifiers, and external calendar item identifiers are included so an AI assistant can relate events to entries editable through tools such as `gog`.

### `POST /sync`

Returns the same weather response as `/weather` for the stored location and schedules configured cron commands for workout and wakeup events. Requires a `full` API key.

JSON body:

```json
{
  "lastWorkout": 1779020000,
  "awake": 1,
  "format": "json"
}
```

Fields:

- `lastWorkout` (int, optional): timestamp-like marker for the most recent workout. The value is treated as an opaque number and only compared to the last stored value. `0` or omitted is treated as "no workout" and is ignored.
- `awake` (optional): `1` when the user is awake, `0` when the user is asleep. The wakeup cron command runs the first time `awake=1` is received after the configured wakeup hour on a date.
- `format` (optional): `json` (default) or `compact`.

`/weather` and `/sync` return an error when they need the stored location and none has been saved.

## API keys

There are two API key types:

- `weather`: can call `/weather` with supplied `lat` and `lon` coordinates.
- `full`: can call all endpoints, use the stored location, update the stored location, and run sync.

## Options file

The service reads configuration from a JSON file. Set the path with `WAKE_OPTIONS_PATH`. Defaults:

- Linux: `/etc/wake-service/options.json`
- macOS: `~/Library/Application Support/wake-service/options.json`

Example:

```json
{
  "userAgent": "Wake/1.0 (you@example.com)",
  "dataDir": "/var/wake-service",
  "apiKeys": [
    {
      "name": "public-weather",
      "key": "weather-key",
      "type": "weather"
    },
    {
      "name": "watch",
      "key": "full-key",
      "type": "full"
    }
  ],
  "cron": {
    "workout": {
      "command": "openclaw cron add --name \"Garmin workout ping\" --delete-after-run --system-event {prompt} --at \"3m\"",
      "prompt": "The user has logged an activity with Garmin. Check Garmin stats and give feedback."
    },
    "wakeup": {
      "command": "openclaw cron add --name \"Wakeup ping\" --delete-after-run --system-event {prompt} --at \"now\"",
      "prompt": "The user is awake. Check current context and help plan the day.",
      "hour": 4
    }
  }
}
```

Notes:

- `userAgent` is required and must be a descriptive identifier for the MET API.
- `apiKeys` is required and every key must have type `weather` or `full`.
- `dataDir` stores Wake's mutable data files: `location.json`, `calendar.json`, and `sync-state.json`. It defaults to the directory containing the options file.
- `cron.workout.command` and `cron.workout.prompt` are required.
- `cron.wakeup.command` and `cron.wakeup.prompt` are required.
- `cron.wakeup.hour` controls the earliest hour of the day that `awake=1` can trigger the wakeup cron command. It defaults to `4`.
- Cron commands are executed through `/bin/sh -c`.
- Wake replaces `{prompt}` in cron commands with the configured prompt, shell-escaped as a single argument.
- Any scheduling delay should be encoded directly in the cron command.
- To target Batty instead of OpenClaw, use a Batty CLI command such as `batty --root /root/github cron add --workspace workout-coach --prompt {prompt} --model openai-codex/gpt-5.4 --thinking medium --in "3m"`.

## Build & Run

```bash
go build ./...
PORT=8080 ./wake-service
```

## Deployment

Deploy your own instance of this service to use it (I don't want your location data). Easily deployed to any server - your favourite AI agent can help you with this!

## AI usage

All code prompted with Codex & reviewed.
