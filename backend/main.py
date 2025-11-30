from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from openai import OpenAI
import os
import random
from pathlib import Path

app = FastAPI(title="mAIrchen API")

# CORS Middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# OpenAI Client für Mistral
client = OpenAI(
    api_key=os.getenv("MISTRAL_API_KEY", "dummy-key"),
    base_url=os.getenv("MISTRAL_BASE_URL", "https://api.mistral.ai/v1")
)

# Grundwortschatz laden
def load_grundwortschatz():
    # gws.md liegt im gleichen Verzeichnis wie main.py
    gws_path = Path(__file__).parent / "gws.md"
    try:
        with open(gws_path, "r", encoding="utf-8") as f:
            return f.read()
    except FileNotFoundError:
        return "Grundwortschatz nicht gefunden"

GRUNDWORTSCHATZ = load_grundwortschatz()

class StoryRequest(BaseModel):
    thema: str
    personen_tiere: str
    ort: str
    stimmung: str
    laenge: int = 10  # Länge in Minuten, Standard: 10

class RandomSuggestions(BaseModel):
    themen: list[str] = [
        "Freundschaft", "Abenteuer", "Zauber", "Tiere im Wald", 
        "Eine Reise", "Ein Geheimnis", "Mut", "Hilfsbereitschaft"
    ]
    personen_tiere: list[str] = [
        "Ein kleiner Hase", "Eine mutige Prinzessin", "Ein frecher Fuchs",
        "Eine weise Eule", "Ein tapferere Ritterin", "Ein tapferer Ritter", 
        "Ein neugieriges Eichhörnchen", "Ein kleines Mädchen", "Ein junger Drache", 
        "Eine zauberhafte Fee", "Ein fröhlicher Bär", "Ein kluger Junge", "Eine singende Nachtigall"
    ]
    orte: list[str] = [
        "im Wald", "am See", "in einem Schloss", "auf einem Bauernhof",
        "in einem verzauberten Garten", "in den Bergen", "am Meer", "in einem Dorf"
    ]
    stimmungen: list[str] = [
        "fröhlich", "spannend", "mysteriös", "lustig",
        "abenteuerlich", "gemütlich", "aufregend", "herzlich"
    ]

@app.get("/")
async def root():
    return {"message": "mAIrchen API - Märchen für Kinder"}

@app.get("/api/random")
async def get_random_suggestions():
    """Gibt zufällige Vorschläge für die Geschichte zurück"""
    suggestions = RandomSuggestions()
    return {
        "thema": random.choice(suggestions.themen),
        "personen_tiere": random.choice(suggestions.personen_tiere),
        "ort": random.choice(suggestions.orte),
        "stimmung": random.choice(suggestions.stimmungen)
    }

@app.post("/api/generate-story")
async def generate_story(request: StoryRequest):
    """Generiert eine Geschichte basierend auf den Eingaben"""
    
    # Berechne Wortanzahl basierend auf Lesezeit
    # Durchschnittliche Lesegeschwindigkeit Kinder: ~80-100 Wörter/Min
    min_words = request.laenge * 80
    max_words = request.laenge * 100
    
    # Prompt erstellen
    prompt = f"""Du bist ein Geschichtenerzähler für Kinder der Klassen 1-4. 
    
Schreibe eine Geschichte mit folgenden Eigenschaften:
- Lesezeit: etwa {request.laenge} Minuten (ca. {min_words}-{max_words} Wörter)
- Thema: {request.thema}
- Personen/Tiere: {request.personen_tiere}
- Ort: {request.ort}
- Stimmung: {request.stimmung}

WICHTIG: Verwende beim Schreiben häufig Wörter aus dem Grundwortschatz der Klassen 1-4 als Leseübung. 
Die Geschichte sollte kindgerecht, spannend und lehrreich sein.

Hier ist der Grundwortschatz zur Orientierung:
{GRUNDWORTSCHATZ[:3000]}

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
        
        # API-Aufruf an Mistral
        response = client.chat.completions.create(
            model=os.getenv("MISTRAL_MODEL", "mistral-small-latest"),
            messages=[
                {"role": "system", "content": "Du bist ein kreativer Geschichtenerzähler für Grundschulkinder."},
                {"role": "user", "content": prompt}
            ],
            temperature=0.8,
            max_tokens=estimated_tokens
        )
        
        content = response.choices[0].message.content
        
        # Parse Titel und Geschichte
        title = "Eine Geschichte"
        story = content
        
        # Suche nach TITEL: im Text (auch wenn es nicht am Anfang steht)
        if "TITEL:" in content:
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
        
        return {
            "success": True,
            "title": title,
            "story": story,
            "parameters": {
                "thema": request.thema,
                "personen_tiere": request.personen_tiere,
                "ort": request.ort,
                "stimmung": request.stimmung,
                "laenge": request.laenge
            }
        }
    
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Fehler beim Generieren der Geschichte: {str(e)}")

@app.get("/health")
async def health_check():
    return {"status": "healthy"}
