# whatdowedo

Suggests activities for a **date**, a **location** and your **preferences**. It
checks the weather forecast, local events and which venues are open that day,
then asks an open-weight LLM to plan a varied day. Tuned for Germany and Europe
but works anywhere Open-Meteo can geocode.

Live: https://whatdowedo.fly.dev

## How it works

```
request ─▶ geocode (Open-Meteo) ─▶ fan-out ─┬─ weather   Open-Meteo            keyless
                                            ├─ events    Ticketmaster Discovery  TICKETMASTER_API_KEY
                                            ├─ venues    Geoapify Places (OSM)   GEOAPIFY_API_KEY
                                            │            Overpass (fallback)     keyless
                                            │            Google Places (New)     GOOGLE_PLACES_API_KEY
                                            └─ merge + opening-hours evaluation
                                                   │
                                        OpenAI-compatible LLM (Groq by default)
                                                   │
                                     JSON suggestions + facts (HTML via htmx or /api)
```

Providers without a key are skipped and reported in the response; nothing fails
hard. Opening hours use a subset of the OSM `opening_hours` syntax to decide
`open | closed | unknown` for the requested weekday. Results are cached in
memory for `CACHE_TTL`.

## Run locally

```sh
cp .env.example .env   # fill in LLM_API_KEY at least
make run               # http://localhost:8080
```

API:

```sh
curl -s localhost:8080/api/suggest -H 'content-type: application/json' \
  -d '{"date":"2026-10-24","location":"Erlangen, Germany","preferences":"kids, rainy-day friendly","tags":["indoor","family"]}'
```

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `LLM_BASE_URL` | `https://api.groq.com/openai/v1` | any OpenAI-compatible endpoint (OpenRouter, Ollama, vLLM) |
| `LLM_API_KEY` | | required for suggestions; without it only facts are returned |
| `LLM_MODEL` | `openai/gpt-oss-120b` | open-weight model id at that endpoint |
| `TICKETMASTER_API_KEY` | | https://developer.ticketmaster.com (5,000 calls/day free) |
| `GEOAPIFY_API_KEY` | | https://myprojects.geoapify.com (3,000 req/day free) |
| `GOOGLE_PLACES_API_KEY` | | Google Cloud, enable "Places API (New)" |
| `PROVIDER_TIMEOUT` | `8s` | per-provider budget |
| `CACHE_TTL` | `10m` | in-memory response cache |
| `VENUE_RADIUS_M` | `5000` | venue search radius |
| `EVENT_RADIUS_KM` | `30` | event search radius |

Getting an LLM key: Groq at https://console.groq.com/keys (free tier, no card).
For OpenRouter set `LLM_BASE_URL=https://openrouter.ai/api/v1` and a model such
as `meta-llama/llama-3.3-70b-instruct`. For local Ollama set
`LLM_BASE_URL=http://localhost:11434/v1` and `LLM_MODEL=llama3.1`.

## Deployment

GitHub Actions (`.github/workflows/ci.yml`) runs tests, builds a multi-arch
image to `ghcr.io/ravio1i/whatdowedo` and deploys that exact digest to fly.io
on every push to `main`.

One-time setup:

```sh
flyctl apps create whatdowedo --org personal
flyctl secrets set LLM_API_KEY=... TICKETMASTER_API_KEY=... GEOAPIFY_API_KEY=... GOOGLE_PLACES_API_KEY=...
flyctl tokens create deploy -x 999999h   # store as repo secret FLY_API_TOKEN
```

The ghcr package must be public (or fly needs registry credentials) for
`flyctl deploy --image` to pull it. Manual deploy: `make deploy`.

## Development

```sh
make test
make docker-run
```

Layout: `cmd/server` (entrypoint), `internal/{geo,weather,events,venues}`
(providers), `internal/llm` (chat client), `internal/suggest` (orchestration,
prompt, cache), `internal/web` (handlers, templates, CSS). No third-party Go
dependencies.
