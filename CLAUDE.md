# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

mAIrchen generates personalized German children's stories ("Märchen") for grades 1-4, using words from the German "Grundwortschatz" (basic vocabulary) as a reading exercise. It's a Go backend + vanilla JS frontend, deployed as a single Docker container (Nginx in front, Go binary behind it, internal only).

## Commands

### Backend (Go)
```bash
cd backend
go mod download                          # install deps
go run main.go                           # run locally (needs AI_PROVIDER env vars, see .env.example)
go test ./pkg/... -v                     # run all tests
go test ./pkg/... -cover                 # with coverage
go test ./pkg/story/... -run TestName -v # run a single test
golangci-lint run                        # lint (matches CI)
```

### Tools (model comparison benchmark)
```bash
cd tools
go run model_comparison.go   # compares story generation across configured models; needs its own .env, see tools/README.md
```

### E2E tests (Playwright)
```bash
cd tests
npm install
npm test          # requires the app already running at http://localhost:80, e.g. with the fake LLM (no API key):
                  #   docker compose -f docker-compose.yml -f docker-compose.e2e.yml up --build -d
npm run test:ui
npm run test:debug
```

### Docker
```bash
cp .env.example .env             # configure AI_PROVIDER etc. first
docker-compose up --build -d     # build and run the single combined container
docker-compose logs -f
docker-compose down
```

### Frontend (static, no build step)
```bash
cd frontend
python -m http.server 8080   # serve locally; expects backend reachable at :8000
```

## Architecture

**Single container, two processes:** Nginx (port 80, public) reverse-proxies `/api/*` to the Go backend, which listens only on `127.0.0.1:8000` and is never exposed directly. This is the core security boundary — see `docker/nginx-combined.conf` and `docker/start-go.sh`.

**Backend package layout** (`backend/pkg/`), each with a single responsibility:
- `config` — reads `AI_PROVIDER` (`openai` | `ollama-cloud` | `ollama-local`) and derives the OpenAI-compatible base URL / model / key. All providers go through the same `go-openai` client since OpenAI, Mistral, Together AI, OpenRouter, and Ollama are all OpenAI-API-compatible. `AI_PROVIDER=openai` respects `OPENAI_BASE_URL` if set (falling back to `api.openai.com`) — this is how the documented Mistral-as-default setup works, so don't hardcode the OpenAI URL there.
- `data` — the Grundwortschatz word list (`gws.md`) is **embedded into the binary at compile time** (`go:embed`), not read from disk at runtime.
- `prompt` — builds system/user prompts from a `StoryRequest`; word-count targets and vocabulary difficulty depend on `Klassenstufe` ("12" vs "34" pulls a different slice of the embedded Grundwortschatz).
- `story` — `Generator.Generate()` makes **one streaming LLM call** and feeds it through `streamParser`: parse the `TITEL:` line (emitted via `OnTitle`), strip markdown line by line, stop at the standalone `ENDE` marker (replaced by a decorative footer), and emit body text via `OnChunk`. The accumulated text is then scanned against the Grundwortschatz dict. Token usage comes from the stream's final usage chunk. Only opening the stream is retried, never after the first chunk.
- `analysis` — matches Grundwortschatz words against generated story text: case-insensitive, Unicode-letter tokens, and a token matches if it *starts with* a dictionary word (so "Hunde" → "Hund"). Entries of ≤3 letters ("Ei", "ab") only match whole tokens.

**`main.go` streams the story as NDJSON** (`title` → `chunk`* → `done` | `error`). HTTP 200 is committed before generation starts, so failures arrive as an in-band `error` event. The request's `model` field is ignored on purpose; the model is always the server-configured one.

**Request-level protection lives in `main.go`**, not in a middleware/package: in-memory (non-persistent) rate limiting per IP, a global daily request cap, and a daily EUR cost budget derived from token usage — all guarded by one `sync.Mutex`. Cost-per-token differs by provider (`ollama-local` is free, others use a rough EUR/1000-token estimate). Because this state is in-memory, it resets on every restart/redeploy — don't treat `/api/stats` numbers as durable.

**`tools/model_comparison.go`** imports the same `backend/pkg/*` packages directly (not via HTTP) so benchmark runs exercise identical prompt-building and story-generation logic as production.

**Frontend is a PWA:** `frontend/sw.js` cache-first-serves the static app shell (HTML/CSS/JS/icons) and always network-only-passes `/api/*` and `/health` — story generation itself still needs the backend, only the UI works offline. Registration (`frontend/sw-register.js`, loaded by both `index.html` and `about.html`) is skipped whenever `navigator.webdriver` is set, so the Playwright suite never runs against a cached app shell. **Bump `CACHE_VERSION` in `sw.js` whenever a cached file's content changes** — otherwise already-installed clients keep serving the stale cached version. `docker/nginx-combined.conf` (and the standalone `frontend/nginx.conf`) exempt `/sw.js` from the `immutable, 1y` static-asset caching applied to other `.js` files, and set the correct MIME type for `/manifest.webmanifest`.

## CI (`.github/workflows/default.yml`)

Runs on every PR/push to `main`, in dependency order: `build` → (`test`, `lint`) → `e2e`. The `e2e` job builds the real Docker image, boots it together with `tests/fakellm` (via `docker-compose.e2e.yml`, a stdlib-only fake OpenAI-compatible server that streams one canned story), waits on `/health`, then runs the Playwright suite against the live container. No API key is needed, and E2E tests the app flow, not real LLM output quality — use `tools/model_comparison.go` for that. If you change the stream format the backend expects (e.g. the `TITEL:`/`ENDE` markers), update `tests/fakellm/main.go` too. `docker-build.yml` pushes the image to `ghcr.io` on push to `main`.
