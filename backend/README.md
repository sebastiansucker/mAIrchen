# mAIrchen - Go Backend

Go-Implementierung des mAIrchen Backends.

## Entwicklung

### Voraussetzungen
- Go 1.21 oder höher
- Docker und Docker Compose (optional)

### Lokal ausführen

```bash
cd backend-go

# Dependencies installieren
go mod download

# Ausführen
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.mistral.ai/v1"
export OPENAI_MODEL="mistral-small-latest"
go run main.go
```

### Mit Docker

```bash
# Build und Start
docker-compose -f docker/docker-compose-go.yml --env-file .env up -d

# Logs anzeigen
docker-compose -f docker/docker-compose-go.yml logs -f

# Stoppen
docker-compose -f docker/docker-compose-go.yml down
```

## Umgebungsvariablen

Siehe `.env.example` für alle verfügbaren Konfigurationsoptionen.

## API Endpoints

- `GET /` - API Info
- `GET /health` - Health Check
- `GET /api/random` - Zufällige Vorschläge
- `GET /api/stats` - Nutzungsstatistiken
- `POST /api/generate-story` - Geschichte generieren

## Features

- ✅ OpenAI-kompatible API (Mistral, OpenAI, Ollama)
- ✅ Rate Limiting (pro IP und global)
- ✅ Cost Tracking
- ✅ Grundwortschatz-Erkennung
- ✅ CORS Support
- ✅ Embedded Grundwortschatz-Datei
- ✅ Strukturiertes Logging
- ✅ Health Checks

## Performance

Das Go-Backend ist deutlich performanter als die Python-Version:
- Schnellerer Start (~0.1s vs ~2s)
- Geringerer Memory-Footprint (~10MB vs ~50MB)
- Bessere Concurrency durch Goroutines
- Kein GIL (Global Interpreter Lock)
