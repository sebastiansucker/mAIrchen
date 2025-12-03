# Model Comparison Tool

Ein Go-Tool zum Vergleichen verschiedener AI-Modelle für die Generierung von Kindergeschichten.

## Features

- ✅ Nutzt die gleiche Logik wie der Hauptserver (geteilte Pakete)
- ✅ Unterstützt mehrere Provider (Ollama Cloud, Ollama Local, Mistral API)
- ✅ Detaillierte Analyse: Wortanzahl, Grundwortschatz, Absätze, Dialoge
- ✅ JSON- und Markdown-Reports

## Installation

```bash
cd tools
go mod tidy
go build -o model_comparison model_comparison.go
```

## Konfiguration

Erstelle eine `.env` Datei im `tools/` Ordner:

```env
# Ollama Cloud
OLLAMA_API_KEY=your_api_key
OLLAMA_MODELS=ministral-3:8b-cloud,mistral:7b

# Ollama Local (optional)
OLLAMA_BASE_URL=http://localhost:11434/v1
OLLAMA_LOCAL_MODELS=mistral:7b,llama3.2:3b

# Mistral API (optional)
OPENAI_API_KEY=your_mistral_api_key
OPENAI_BASE_URL=https://api.mistral.ai/v1
MISTRAL_MODELS=mistral-small-latest,mistral-large-latest
```

## Verwendung

```bash
./model_comparison
```

Die Ergebnisse werden in `test_results/` gespeichert:
- `full_results_TIMESTAMP.json` - Detaillierte JSON-Ergebnisse
- `full_report_TIMESTAMP.md` - Markdown-Report
- `latest_full_report.md` - Immer der neueste Report

## Test-Cases

Das Tool testet jedes Modell mit 5 verschiedenen Szenarien:
- Klasse 1-2: Einfache Tiergeschichte (2 Minuten)
- Klasse 1-2: Freundschaftsgeschichte (3 Minuten)
- Klasse 3-4: Freundschaftsgeschichte (3 Minuten)
- Klasse 3-4: Abenteuergeschichte (5 Minuten)
- Klasse 3-4: Zaubergeschichte (3 Minuten)

## Architektur

Das Tool importiert die Pakete aus `backend-go/pkg/`:
- `config` - Provider-Konfiguration
- `prompt` - Prompt-Generierung
- `story` - Story-Generierung
- `analysis` - Grundwortschatz-Analyse

Dadurch wird sichergestellt, dass die gleiche Logik wie im Produktionsserver verwendet wird.
