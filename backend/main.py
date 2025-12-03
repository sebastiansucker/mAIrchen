from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, field_validator
from openai import OpenAI
import os
import random
from pathlib import Path
from datetime import datetime, timedelta
from collections import defaultdict
import threading
import json
import logging
import traceback

# Logging konfigurieren
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = FastAPI(title="mAIrchen API")

# Rate Limiting Konfiguration
RATE_LIMIT_PER_IP = int(os.getenv("RATE_LIMIT_PER_IP", "10"))  # Anfragen pro Stunde
RATE_LIMIT_WINDOW = 3600  # 1 Stunde in Sekunden
GLOBAL_DAILY_LIMIT = int(os.getenv("GLOBAL_DAILY_LIMIT", "1000"))  # Max Anfragen pro Tag
MAX_STORY_LENGTH = int(os.getenv("MAX_STORY_LENGTH", "15"))  # Max Minuten

# Cost Monitoring
MAX_DAILY_COST = float(os.getenv("MAX_DAILY_COST", "5.0"))  # Max 5€ pro Tag
COST_PER_REQUEST = 0.0015  # Geschätzte Kosten pro Anfrage (wird dynamisch angepasst)

# Rate Limiting Storage (In-Memory)
request_history = defaultdict(list)  # IP -> [timestamp, timestamp, ...]
global_request_count = {"count": 0, "reset_time": datetime.now() + timedelta(days=1)}
daily_cost = {"cost": 0.0, "reset_time": datetime.now() + timedelta(days=1)}
rate_limit_lock = threading.Lock()

# CORS Middleware - nur für gleiche Domain (über Nginx Proxy)
# In Produktion sollte hier die tatsächliche Domain stehen
allowed_origins = os.getenv(
    "ALLOWED_ORIGINS",
    "http://localhost,http://localhost:80,http://localhost:8080"
).split(",")

app.add_middleware(
    CORSMiddleware,
    allow_origins=allowed_origins,
    allow_credentials=True,
    allow_methods=["POST", "GET"],
    allow_headers=["Content-Type"],
)

# AI Provider Konfiguration
AI_PROVIDER = os.getenv("AI_PROVIDER", "openai").lower()  # openai, ollama-cloud, ollama-local

# Client initialisierung basierend auf Provider
if AI_PROVIDER == "ollama-cloud":
    # Ollama Cloud mit API Key
    client = OpenAI(
        api_key=os.getenv("OLLAMA_API_KEY", "dummy-key"),
        base_url="https://ollama.com/v1"
    )
    DEFAULT_MODEL = os.getenv("OLLAMA_MODEL", "llama3.2:3b")
elif AI_PROVIDER == "ollama-local":
    # Lokale Ollama Instanz (kein API Key nötig)
    client = OpenAI(
        api_key="ollama",  # Dummy key für lokale Instanz
        base_url=os.getenv("OLLAMA_BASE_URL", "http://localhost:11434/v1")
    )
    DEFAULT_MODEL = os.getenv("OLLAMA_MODEL", "llama3.2:3b")
else:
    # OpenAI-compatible API (default)
    client = OpenAI(
        api_key=os.getenv("OPENAI_API_KEY", "dummy-key"),
        base_url=os.getenv("OPENAI_BASE_URL", "https://api.mistral.ai/v1")
    )
    DEFAULT_MODEL = os.getenv("OPENAI_MODEL", "mistral-small-latest")

# Grundwortschatz laden
def load_grundwortschatz():
    # gws.md liegt im gleichen Verzeichnis wie main.py
    gws_path = Path(__file__).parent / "gws.md"
    try:
        with open(gws_path, "r", encoding="utf-8") as f:
            return f.read()
    except FileNotFoundError:
        return "Grundwortschatz nicht gefunden"

# Grundwortschatz für alle Klassen laden
GRUNDWORTSCHATZ_FULL = load_grundwortschatz()

# Grundwortschatz für Klasse 1/2 extrahieren (bis zur Zeile mit "Jahrgangsstufen 3 und 4")
def load_grundwortschatz_12():
    gws_path = Path(__file__).parent / "gws.md"
    try:
        with open(gws_path, "r", encoding="utf-8") as f:
            content = f.read()
            # Schneide bei "Jahrgangsstufen 3 und 4" ab
            parts = content.split("### **Grundwortschatz für Jahrgangsstufen 3 und 4**")
            return parts[0] if len(parts) > 0 else content
    except FileNotFoundError:
        return "Grundwortschatz nicht gefunden."

GRUNDWORTSCHATZ_12 = load_grundwortschatz_12()

class StoryRequest(BaseModel):
    thema: str
    personen_tiere: str
    ort: str
    stimmung: str
    laenge: int = 10  # Länge in Minuten, Standard: 10
    klassenstufe: str = "34"  # "12" oder "34", Standard: 3/4 Klasse
    stil: str = ""  # Stil/Genre der Geschichte
    
    @field_validator('laenge')
    @classmethod
    def validate_laenge(cls, v):
        if v < 1:
            raise ValueError('Länge muss mindestens 1 Minute sein')
        if v > MAX_STORY_LENGTH:
            raise ValueError(f'Länge darf maximal {MAX_STORY_LENGTH} Minuten sein')
        return v
    
    @field_validator('thema', 'personen_tiere', 'ort', 'stimmung', 'stil')
    @classmethod
    def validate_string_length(cls, v):
        if len(v) > 200:
            raise ValueError('Eingabe zu lang (max 200 Zeichen)')
        return v
    
    @field_validator('klassenstufe')
    @classmethod
    def validate_klassenstufe(cls, v):
        if v not in ['12', '34']:
            raise ValueError('Klassenstufe muss "12" oder "34" sein')
        return v

class RandomSuggestions(BaseModel):
    themen: list[str] = [
        "Freundschaft", "Abenteuer", "Zauber", "Tiere im Wald", 
        "Eine Reise", "Ein Geheimnis", "Mut", "Hilfsbereitschaft"
    ]
    personen_tiere: list[str] = [
        "Ein kleiner Hase namens Erwin", "Eine mutige Prinzessin namens Helena", "Ein frecher Fuchs namens Felix",
        "Eine weise Eule", "Ein tapferere Ritterin names Hannelore", "Ein tapferer Ritter names Siegfried", 
        "Ein neugieriges Eichhörnchen", "Ein kleines Mädchen namens Juna", "Ein junger Drache", 
        "Eine zauberhafte Fee", "Der fröhliche Bär Klaus", "Ein kluger Junge", "Eine singende Nachtigall"
    ]
    orte: list[str] = [
        "im Wald", "am See", "in einem Schloss", "auf einem Bauernhof",
        "in einem verzauberten Garten", "in den Bergen", "am Meer", "in einem Dorf"
    ]
    stimmungen: list[str] = [
        "fröhlich", "spannend", "mysteriös", "lustig",
        "abenteuerlich", "gemütlich", "aufregend", "herzlich"
    ]
    stile: list[str] = [
        "Michael Ende", "Marc-Uwe Kling", "Astrid Lindgren", "Janosch",
        "Cornelia Funke", "Märchen", "Fabel", "Moderne Kindergeschichte"
    ]

@app.get("/")
async def root():
    return {
        "message": "mAIrchen API - Märchen für Kinder",
        "ai_provider": AI_PROVIDER,
        "model": DEFAULT_MODEL
    }

def get_client_ip(request: Request) -> str:
    """Extrahiert die Client-IP aus dem Request (berücksichtigt Proxy)"""
    forwarded = request.headers.get("X-Forwarded-For")
    if forwarded:
        return forwarded.split(",")[0].strip()
    return request.client.host if request.client else "unknown"

def check_rate_limit(ip: str) -> tuple[bool, str]:
    """Prüft Rate Limit für eine IP. Returns (allowed, error_message)"""
    with rate_limit_lock:
        now = datetime.now()
        
        # Reset global counter täglich
        if now > global_request_count["reset_time"]:
            global_request_count["count"] = 0
            global_request_count["reset_time"] = now + timedelta(days=1)
        
        # Reset daily cost täglich
        if now > daily_cost["reset_time"]:
            daily_cost["cost"] = 0.0
            daily_cost["reset_time"] = now + timedelta(days=1)
        
        # Prüfe tägliches Budget
        if daily_cost["cost"] >= MAX_DAILY_COST:
            hours_until_reset = (daily_cost["reset_time"] - now).seconds // 3600
            return False, f"Tägliches Budget erreicht. Service pausiert für ~{hours_until_reset}h."
        
        # Prüfe globales Limit
        if global_request_count["count"] >= GLOBAL_DAILY_LIMIT:
            hours_until_reset = (global_request_count["reset_time"] - now).seconds // 3600
            return False, f"Tägliches Anfrage-Limit erreicht. Bitte in ~{hours_until_reset}h erneut versuchen."
        
        # Bereinige alte Requests (älter als RATE_LIMIT_WINDOW)
        cutoff_time = now - timedelta(seconds=RATE_LIMIT_WINDOW)
        request_history[ip] = [ts for ts in request_history[ip] if ts > cutoff_time]
        
        # Prüfe IP-spezifisches Limit
        if len(request_history[ip]) >= RATE_LIMIT_PER_IP:
            minutes_until_oldest_expires = int((request_history[ip][0] + timedelta(seconds=RATE_LIMIT_WINDOW) - now).seconds / 60)
            return False, f"Zu viele Anfragen. Bitte warte ~{minutes_until_oldest_expires} Minuten."
        
        # Request erlauben und zählen
        request_history[ip].append(now)
        global_request_count["count"] += 1
        daily_cost["cost"] += COST_PER_REQUEST
        
        return True, ""

@app.get("/api/random")
async def get_random_suggestions():
    """Gibt zufällige Vorschläge für die Geschichte zurück"""
    suggestions = RandomSuggestions()
    return {
        "thema": random.choice(suggestions.themen),
        "personen_tiere": random.choice(suggestions.personen_tiere),
        "ort": random.choice(suggestions.orte),
        "stimmung": random.choice(suggestions.stimmungen),
        "stil": random.choice(suggestions.stile)
    }

@app.get("/api/stats")
async def get_stats():
    """Gibt aktuelle Nutzungsstatistiken zurück (nur für Monitoring)"""
    with rate_limit_lock:
        return {
            "global_requests_today": global_request_count["count"],
            "global_limit": GLOBAL_DAILY_LIMIT,
            "estimated_cost_today": round(daily_cost["cost"], 2),
            "daily_budget": MAX_DAILY_COST,
            "budget_remaining": round(MAX_DAILY_COST - daily_cost["cost"], 2),
            "rate_limit_per_ip": RATE_LIMIT_PER_IP,
            "active_ips": len(request_history)
        }

@app.post("/api/generate-story")
async def generate_story(story_request: StoryRequest, request: Request):
    """Generiert eine Geschichte basierend auf den Eingaben"""
    
    logger.info(f"Story-Generierung gestartet - IP: {get_client_ip(request)}")
    logger.debug(f"Request-Parameter: {story_request.model_dump()}")
    
    # Rate Limiting prüfen
    client_ip = get_client_ip(request)
    allowed, error_msg = check_rate_limit(client_ip)
    if not allowed:
        logger.warning(f"Rate Limit erreicht für IP {client_ip}: {error_msg}")
        raise HTTPException(status_code=429, detail=error_msg)
    
    # Berechne Wortanzahl basierend auf Lesezeit
    # Durchschnittliche Lesegeschwindigkeit abhängig von Klassenstufe
    if story_request.klassenstufe == "12":
        # Klasse 1 & 2: ~70 Wörter/Min
        min_words = story_request.laenge * 60
        max_words = story_request.laenge * 70
    else:
        # Klasse 3 & 4: ~80-100 Wörter/Min
        min_words = story_request.laenge * 80
        max_words = story_request.laenge * 100
    
    # Wähle passenden Grundwortschatz und Schwierigkeitsgrad
    if story_request.klassenstufe == "12":
        grundwortschatz = GRUNDWORTSCHATZ_12  # Kompletter Grundwortschatz für Klasse 1&2
        zielgruppe = "Kinder der Klassenstufen 1 & 2"
        schwierigkeit = "sehr einfach mit kurzen Sätzen und einfachen Wörtern"
    else:
        grundwortschatz = GRUNDWORTSCHATZ_FULL  # Kompletter Grundwortschatz für alle Klassen
        zielgruppe = "Kinder der Klassenstufen 3 & 4"
        schwierigkeit = "kindgerecht mit etwas längeren Sätzen und anspruchsvolleren Wörtern"
    
    # Prompt erstellen
    stil_instruction = ""
    if story_request.stil:
        stil_instruction = f"- Stil/Genre: Schreibe im Stil von '{story_request.stil}' oder als {story_request.stil}\n"
    
    prompt = f"""Du bist ein Geschichtenerzähler für {zielgruppe}. 
    
Schreibe eine Geschichte mit folgenden Eigenschaften:
- Lesezeit: etwa {story_request.laenge} Minuten (ca. {min_words}-{max_words} Wörter)
- Thema: {story_request.thema}
- Personen/Tiere: {story_request.personen_tiere}
- Ort: {story_request.ort}
- Stimmung: {story_request.stimmung}
{stil_instruction}- Schwierigkeitsgrad: {schwierigkeit}

WICHTIG: Verwende beim Schreiben häufig Wörter aus dem Grundwortschatz als Leseübung. 
Die Geschichte sollte kindgerecht, spannend und lehrreich sein.

Hier ist der Grundwortschatz zur Orientierung:
{grundwortschatz}

Format:
Gib die Antwort im folgenden Format zurück:
TITEL: [Ein kurzer, ansprechender Titel für die Geschichte]

[Die Geschichte in Absätzen]

Beginne direkt mit "TITEL:" gefolgt vom Titel.

WICHTIG: Schreibe wirklich die vollständige Geschichte mit ca. {max_words} Wörtern. Mache die Geschichte nicht kürzer!"""

    try:
        # Berechne max_tokens basierend auf gewünschter Länge
        # ~1.3 Tokens pro Wort für Deutsch, plus Buffer für Titel/Formatierung
        estimated_tokens = int(max_words * 1.3) + 200
        
        logger.debug(f"AI Provider: {AI_PROVIDER}, Model: {DEFAULT_MODEL}")
        logger.debug(f"Estimated tokens: {estimated_tokens}")
        
        # API-Aufruf an AI Provider
        logger.info("Starte API-Aufruf...")
        response = client.chat.completions.create(
            model=DEFAULT_MODEL,
            messages=[
                {"role": "system", "content": "Du bist ein kreativer Geschichtenerzähler für Grundschulkinder."},
                {"role": "user", "content": prompt}
            ],
            temperature=0.8,
            max_tokens=estimated_tokens
        )
        
        logger.info("API-Aufruf erfolgreich")
        content = response.choices[0].message.content
        logger.debug(f"Response Länge: {len(content) if content else 0} Zeichen")
        
        # Parse Titel und Geschichte
        title = "Eine Geschichte"
        story = content if content else ""
        
        # Suche nach TITEL: im Text (auch wenn es nicht am Anfang steht)
        if content and "TITEL:" in content:
            parts = content.split("TITEL:", 1)
            if len(parts) > 1:
                # Extrahiere Titel (erste Zeile nach TITEL:)
                rest = parts[1].strip()
                title_end = rest.find("\n")
                if title_end > 0:
                    title = rest[:title_end].strip()
                    story = rest[title_end+1:].strip()
                else:
                    title = rest.strip()
                    story = ""
        
        # Fallback: Wenn der Titel noch "**TITEL:" enthält, entferne die Markdown-Sterne
        title = title.replace("**", "").strip()
        
        # Aktualisiere Cost Tracking basierend auf tatsächlichem Token-Verbrauch
        if hasattr(response, 'usage') and response.usage:
            total_tokens = response.usage.total_tokens
            # Provider-spezifisches Pricing
            if AI_PROVIDER == "ollama-cloud":
                # Ollama Cloud: ~0.0005€ per 1K tokens
                actual_cost = (total_tokens / 1000) * 0.0005
            elif AI_PROVIDER == "ollama-local":
                # Lokale Ollama: kostenlos
                actual_cost = 0.0
            else:
                # OpenAI-compatible: ~0.001€ per 1K tokens (adjust based on provider)
                actual_cost = (total_tokens / 1000) * 0.001
            
            with rate_limit_lock:
                daily_cost["cost"] += actual_cost
        
        return {
            "success": True,
            "title": title,
            "story": story,
            "parameters": {
                "thema": story_request.thema,
                "personen_tiere": story_request.personen_tiere,
                "ort": story_request.ort,
                "stimmung": story_request.stimmung,
                "stil": story_request.stil,
                "laenge": story_request.laenge,
                "klassenstufe": story_request.klassenstufe
            }
        }
    
    except Exception as e:
        logger.error(f"Fehler beim Generieren der Geschichte: {str(e)}")
        logger.error(f"Traceback: {traceback.format_exc()}")
        logger.error(f"AI Provider: {AI_PROVIDER}, Model: {DEFAULT_MODEL}")
        logger.error(f"API Key gesetzt: {bool(os.getenv('OPENAI_API_KEY') or os.getenv('OLLAMA_API_KEY'))}")
        raise HTTPException(status_code=500, detail=f"Fehler beim Generieren der Geschichte: {str(e)}")

@app.get("/health")
async def health_check():
    return {"status": "healthy"}
