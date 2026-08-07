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
npm test          # requires the app already running at http://localhost:80 (docker compose up)
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
- `story` — `Generator.Generate()` orchestrates the full flow: call the LLM → strip markdown the model may have added anyway → parse out `TITEL:`/`ENDE` markers → run a **second LLM call** that only fixes spelling (temperature 0.1, told explicitly not to touch content/style) → re-strip markdown → scan the result against the Grundwortschatz dict to report which vocabulary words appear. Token usage from both calls is summed and returned for cost tracking.
- `analysis` — matches Grundwortschatz words against generated story text (case-insensitive, word-boundary regex).

**Request-level protection lives in `main.go`**, not in a middleware/package: in-memory (non-persistent) rate limiting per IP, a global daily request cap, and a daily EUR cost budget derived from token usage — all guarded by one `sync.Mutex`. Cost-per-token differs by provider (`ollama-local` is free, others use a rough EUR/1000-token estimate). Because this state is in-memory, it resets on every restart/redeploy — don't treat `/api/stats` numbers as durable.

**`tools/model_comparison.go`** imports the same `backend/pkg/*` packages directly (not via HTTP) so benchmark runs exercise identical prompt-building and story-generation logic as production.

## CI (`.github/workflows/default.yml`)

Runs on every PR/push to `main`, in dependency order: `build` → (`test`, `lint`) → `e2e`. The `e2e` job builds the real Docker image, boots it with `AI_PROVIDER=ollama-cloud` (needs the `OLLAMA_API_KEY` repo secret), waits on `/health`, then runs the Playwright suite against the live container — so E2E failures can mean either a real bug or a missing/expired secret. `docker-build.yml` pushes the image to `ghcr.io` on push to `main`.
