# 📚 mAIrchen - Geschichten für Kinder

Eine Märchen-Schreib-App für Grundschulkinder (Klasse 1-4), die personalisierte Geschichten mit Wörtern aus dem Grundwortschatz generiert.

## ✨ Funktionen

- **Personalisierte Geschichten**: Der Nutzer gibt Thema, Personen/Tiere, Ort und Stimmung ein
- **Zufalls-Generator**: Automatische Vorschläge für alle Parameter
- **Grundwortschatz-Integration**: Geschichten enthalten Wörter aus dem Grundwortschatz der Klassen 1-4
- **Buchlayout**: Ansprechende Darstellung im Buchformat für optimales Leseerlebnis
- **KI-gestützt**: Nutzt Mistral AI über OpenAI-kompatible API

## 🚀 Installation

### Voraussetzungen

- Docker und Docker Compose
- Mistral API Key

### Setup

1. Repository klonen und in das Verzeichnis wechseln:
```bash
cd mAIrchen
```

2. Umgebungsvariablen konfigurieren:
```bash
cp .env.example .env
```

3. `.env`-Datei bearbeiten und Mistral API Key eintragen:
```
MISTRAL_API_KEY=your-actual-api-key
```

4. Container starten:
```bash
docker-compose -f docker/docker-compose.yml build && docker-compose --env-file .env -f docker/docker-compose.yml up -d
```

Die App ist nun verfügbar unter:
- **Frontend**: http://localhost
- **Backend API**: http://localhost:8000
- **API Dokumentation**: http://localhost:8000/docs

## 🏗️ Architektur

### Backend (FastAPI)
- Python-basierte REST API
- OpenAI-kompatibler Client für Mistral
- Endpunkte:
  - `GET /api/random` - Zufällige Vorschläge
  - `POST /api/generate-story` - Geschichte generieren
  - `GET /health` - Health Check

### Frontend
- Vanilla HTML/CSS/JavaScript
- Responsive Design
- Buchlayout für optimale Leseerfahrung
- Nginx als Webserver

### Dateien
```
mAIrchen/
├── backend/
│   ├── main.py           # FastAPI Backend
│   └── requirements.txt  # Python Dependencies
├── frontend/
│   ├── index.html       # Haupt-HTML
│   ├── styles.css       # Styling
│   └── app.js           # JavaScript Logik
├── gws.md               # Grundwortschatz
├── docker-compose.yml   # Container Orchestrierung
├── Dockerfile.backend   # Backend Container
├── Dockerfile.frontend  # Frontend Container
├── nginx.conf          # Nginx Konfiguration
└── .env.example        # Umgebungsvariablen Template
```

## 🎯 Verwendung

1. App im Browser öffnen (http://localhost)
2. Eingabefelder ausfüllen:
   - Thema (z.B. "Freundschaft")
   - Personen/Tiere (z.B. "Ein kleiner Hase")
   - Ort (z.B. "im Wald")
   - Stimmung (z.B. "fröhlich")
3. Optional: "🎲 Zufällig" Button für automatische Vorschläge
4. "✨ Geschichte erstellen" klicken
5. Geschichte im Buchlayout lesen

## 🛠️ Entwicklung

### Backend lokal starten
```bash
cd backend
pip install -r requirements.txt
uvicorn main:app --reload
```

### Frontend lokal testen
Einfach `frontend/index.html` in einem Browser öffnen oder mit einem lokalen Webserver:
```bash
cd frontend
python -m http.server 8080
```

## 📝 API Endpunkte

### Zufällige Vorschläge
```http
GET /api/random
```

### Geschichte generieren
```http
POST /api/generate-story
Content-Type: application/json

{
  "thema": "Abenteuer",
  "personen_tiere": "Ein mutiger Fuchs",
  "ort": "im Wald",
  "stimmung": "spannend"
}
```

## 🔧 Konfiguration

Umgebungsvariablen in `.env`:
- `MISTRAL_API_KEY`: Ihr Mistral API Schlüssel
- `MISTRAL_BASE_URL`: API Basis-URL (Standard: https://api.mistral.ai/v1)
- `MISTRAL_MODEL`: Zu verwendendes Modell (Standard: mistral-small-latest)

## 📄 Lizenz

Privates Projekt für Bildungszwecke.