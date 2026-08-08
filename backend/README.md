# mAIrchen - Go Backend

Go-Implementierung des mAIrchen Backends.

## Entwicklung

### Voraussetzungen
- Go 1.25 oder höher (CI/Docker-Build nutzen 1.26)
- Docker und Docker Compose (optional)

### Lokal ausführen

```bash
cd backend

# Dependencies installieren
go mod download

# Ausführen
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.mistral.ai/v1"
export OPENAI_MODEL="mistral-small-latest"
go run main.go
```

### Mit Docker

Vom Repository-Root aus (nicht aus `backend/`):

```bash
# Build und Start
docker-compose up --build -d

# Logs anzeigen
docker-compose logs -f

# Stoppen
docker-compose down
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

- Schneller Start (~0.1s)
- Geringer Memory-Footprint (~10MB)
- Gute Concurrency durch Goroutines
