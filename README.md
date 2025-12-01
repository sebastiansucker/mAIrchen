# 📚 mAIrchen - Geschichten für Kinder

Eine Märchen-Schreib-App für Grundschulkinder (Klasse 1-4), die personalisierte Geschichten mit Wörtern aus dem Grundwortschatz generiert.

## ✨ Funktionen

- **Personalisierte Geschichten**: Der Nutzer gibt Thema, Personen/Tiere, Ort und Stimmung ein
- **Zufalls-Generator**: Automatische Vorschläge für alle Parameter
- **Grundwortschatz-Integration**: Geschichten enthalten Wörter aus dem Grundwortschatz der Klassen 1-4
- **Buchlayout**: Ansprechende Darstellung im Buchformat mit vergilbtem Papier-Look
- **Seitenblätter-Animation**: Geschichten erscheinen mit einer 3D-Blätter-Animation
- **KI-gestützt**: Nutzt Mistral AI über OpenAI-kompatible API
- **Single-Container**: Frontend und Backend in einem Container für einfaches Deployment

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

4. Container bauen & starten:
```bash
docker-compose --env-file .env -f docker/docker-compose.yml build
docker-compose --env-file .env -f docker/docker-compose.yml up -d
```

Die App ist nun verfügbar unter:
- **Frontend**: http://localhost
- **API**: http://localhost/api/
- **Health Check**: http://localhost/health

## 🏗️ Architektur

### Single Container Setup
Frontend und Backend laufen in einem Docker-Container:
- **Nginx** serviert das Frontend (Port 80)
- **FastAPI** Backend läuft auf Port 8000 (intern)
- Nginx fungiert als Reverse Proxy für `/api/*` Requests

### Backend (FastAPI)
- Python-basierte REST API
- OpenAI-kompatibler Client für Mistral
- Grundwortschatz-Integration aus `backend/gws.md`
- Endpunkte:
  - `GET /api/random` - Zufällige Vorschläge
  - `POST /api/generate-story` - Geschichte generieren
  - `GET /health` - Health Check

### Frontend
- Vanilla HTML/CSS/JavaScript
- Responsive Design
- 3D-Seitenblätter-Animation
- Buchlayout mit vergilbtem Papier-Effekt
- Automatische API-URL-Erkennung (funktioniert im Netzwerk)

### Projektstruktur
```
mAIrchen/
├── .env.example          # Umgebungsvariablen Template
├── .gitignore           # Git Ignore Datei
├── README.md            # Diese Datei
├── backend/
│   ├── main.py          # FastAPI Backend
│   ├── requirements.txt # Python Dependencies
│   └── gws.md          # Grundwortschatz Klassen 1-4
├── frontend/
│   ├── index.html      # Haupt-HTML
│   ├── styles.css      # Styling & Animationen
│   ├── app.js          # JavaScript Logik
│   ├── logo.png        # App Logo (transparent)
│   └── app_icon.png    # App Icon
└── docker/
    ├── Dockerfile              # Multi-Stage Build
    ├── docker-compose.yml      # Container Orchestrierung
    └── nginx-combined.conf     # Nginx Konfiguration
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
export MISTRAL_API_KEY=your-key
uvicorn main:app --reload --host 0.0.0.0 --port 8000
```

### Frontend lokal testen
Das Frontend benötigt das Backend auf Port 8000:
```bash
cd frontend
python -m http.server 8080
```
Dann im Browser: http://localhost:8080

### Container neu bauen nach Änderungen
```bash
docker-compose --env-file .env -f docker/docker-compose.yml build
docker-compose --env-file .env -f docker/docker-compose.yml up -d
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
- `MISTRAL_API_KEY`: Ihr Mistral API Schlüssel (erforderlich)
- `MISTRAL_BASE_URL`: API Basis-URL (Standard: https://api.mistral.ai/v1)
- `MISTRAL_MODEL`: Zu verwendendes Modell (Standard: mistral-small-latest)
- `ALLOWED_ORIGINS`: Erlaubte CORS Origins (Standard: http://localhost,http://localhost:80)

**Wichtig**: Die `.env` Datei ist in `.gitignore` und wird nicht ins Repository committed!

## 🔒 Sicherheit

Die API ist durch mehrere Sicherheitsebenen geschützt:

1. **Backend nur auf localhost**: Das FastAPI-Backend lauscht nur auf `127.0.0.1:8000` und ist von außen nicht direkt erreichbar
2. **Nginx als einziger Zugangspunkt**: Nur Nginx kann auf das Backend zugreifen und fungiert als Reverse Proxy
3. **CORS-Einschränkung**: Nur erlaubte Origins (konfiguriert via `ALLOWED_ORIGINS`) können API-Requests durchführen

### Für Produktion

In der Produktion sollten Sie `ALLOWED_ORIGINS` auf Ihre echte Domain(s) setzen:

```bash
# In .env
ALLOWED_ORIGINS=https://mairchen.de,https://www.mairchen.de
```

Dies verhindert, dass andere Websites Ihre API nutzen können, auch wenn sie die URL kennen. Das Frontend kann weiterhin von Client-Geräten auf die API zugreifen, da die Requests über Ihren Server laufen.

## 🌐 Netzwerk-Zugriff

Die App ist von anderen Geräten im Netzwerk erreichbar:
1. Finde die IP-Adresse deines Computers: `ifconfig` (Mac/Linux) oder `ipconfig` (Windows)
2. Öffne auf einem anderen Gerät: `http://<deine-ip>`

Das Frontend nutzt automatisch die richtige URL für API-Requests.

## 🐳 Deployment

### Mit GitHub Container Registry
Der Container wird automatisch bei jedem Push auf `main` gebaut und in die GitHub Container Registry gepusht.

**Container direkt von GitHub pullen:**
```bash
docker pull ghcr.io/sebastiansucker/mairchen:latest
docker run -d -p 80:80 \
  -e MISTRAL_API_KEY=your-key \
  -e MISTRAL_BASE_URL=https://api.mistral.ai/v1 \
  -e MISTRAL_MODEL=mistral-small-latest \
  --name mairchen-app \
  ghcr.io/sebastiansucker/mairchen:latest
```

**Mit Docker Compose und GitHub Registry:**
```yaml
services:
  app:
    image: ghcr.io/sebastiansucker/mairchen:latest
    container_name: mairchen-app
    ports:
      - "80:80"
    environment:
      - MISTRAL_API_KEY=${MISTRAL_API_KEY}
      - MISTRAL_BASE_URL=${MISTRAL_BASE_URL:-https://api.mistral.ai/v1}
      - MISTRAL_MODEL=${MISTRAL_MODEL:-mistral-small-latest}
    restart: unless-stopped
```

### Manuelles Deployment (Lokaler Build)
```bash
# Auf dem Server
git clone git@github.com:sebastiansucker/mAIrchen.git
cd mAIrchen
cp .env.example .env
# .env bearbeiten und API-Key eintragen
docker-compose --env-file .env -f docker/docker-compose.yml up -d
```