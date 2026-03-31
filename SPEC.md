This project is for a new service, Wake, that will replace and extend Yrproxy.

Source and documentation for Yrproxy: /Users/dt/Github/undertow-watchface/proxy

We will add a regular JSON format to the output in addition to the existing compact format for Garmin. Add GET parameter format=json or compact, default json. The purpose of this is to make the output readable by an AI agent in addition to its current use case. The JSON format should mirror the compact fields, but each forecast entry is an object with named keys instead of array indexes (e.g. time, seaTemperature, waveHeight, waveDirection, temperature, windSpeed, windDirection, cloudCover, condition, uvIndex, precipitation12hours). In JSON, cloudCover should be an object with total/low/medium/high fields and time should be formatted in local date/time. Include a meta.units object describing units in JSON output. The compact format should omit cloud cover and stay at 12 entries, while JSON should return all available forecast times. lat/lon must be provided; no default coordinates.

The interpretation of the weather should move from the Garmin watch face to the server. That means grouping of weather symbols, interpretation of clouds for new symbols (heavy clouds, high clouds, levels of partly cloud). However do keep the distinction between weather types that map to snow (eg. hail) and to thunder (eg. storm I think). Grouping of those should still be on the watch. Do not keep backward compatibility with the existing watch output; the watch will be updated later.

Read the watchface implementation for reference but do not fix the watchface to match the server yet: 
/Users/dt/Github/undertow-watchface/Undertow

The wake service will keep the proxys feature to log the location to a file, but only when called from the watch. We will set API keys in the options file where location logging can be enabled or disabled for each key (treat those keys as the watch callers). Add a logLocation query param to /weather to trigger logging when allowed.

Add caching of forecasts. Forecasts should be cached for one hour with the GPS coordinates (four decimals) as key. Make sure a cached entry can be used to return both formats.

The user agent should be read from the options file; fail if this is not set. Make sure not to check the current user agent into the repo.

We will add a feature so that the watch can ping an OpenClaw instance when logging an activity. Schedule the OpenClaw cron with a configurable delay (default 3 minutes). This allows the workout to sync to Garmin Connect first. Have a configurable prompt for this, default "The user has logged an activity with Garmin. Check Garmin stats and give feedback." Example for pinging OpenClaw:

curl -X POST [http://localhost:18789/api/cron/add](http://localhost:18789/api/cron/add) \
-H "Authorization: Bearer $TOKEN" \
-H "Content-Type: application/json" \
-d '{
"job": {
"name": "Garmin activity feedback",
"schedule": { "kind": "at", "at": "'$(date -d "+3 minutes" +%s)000'" },
"sessionTarget": "main",
"wakeMode": "now",
"payload": { "kind": "systemEvent", "text": "🏃 Ny aktivitet logget - tjek Garmin og giv feedback!" },
"deleteAfterRun": true
}
}'

The token and URL for OpenClaw should also be in the options file.

Have the weather forecast at /weather and add a /workout endpoint for this feature. /workout should be a POST that accepts a JSON body with the activity count for the day (number). Return a simple ok/error response (no need to proxy the OpenClaw response). Also have workout be an option for each API key.

Add a readme file with documentation on these features and the options file. Give credit to YR for the weather data.

Read config file path from an env var, default path should be something reasonable on Mac and Linux (Linux most important).

The current implementation is in Go - we can keep that choice but it's not a requirement. Priorities are easy deployment on a VPS, readability and it should be good for vibe coding.

Do build and test the service. We have Go installed here. Commit and push once it's working.
