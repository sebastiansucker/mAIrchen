#!/usr/bin/env python3
"""
Test-Script zum vollständigen Vergleich aller Modelle (Ollama lokal + Mistral API).

Usage:
    python full_comparison.py

Testet alle verfügbaren Modelle aus beiden Quellen:
- Ollama lokale Modelle
- Mistral API Modelle

Bewertet:
- Geschwindigkeit (Generierungszeit)
- Story-Länge (Wortanzahl)
- Grundwortschatz-Nutzung
- Struktur (Absätze, Dialoge)
- Kindgerechte Sprache
"""

import os
import sys
import time
import json
from datetime import datetime
from pathlib import Path
import re
import requests
from dotenv import load_dotenv

# Lade Umgebungsvariablen aus .env im tools-Ordner
env_path = Path(__file__).parent / ".env"
load_dotenv(env_path)

# API Client
try:
    from openai import OpenAI
except ImportError:
    print("❌ OpenAI library nicht installiert. Bitte ausführen: pip install openai")
    sys.exit(1)

# Grundwortschatz laden
def load_grundwortschatz():
    """Lädt den kompletten Grundwortschatz aus gws.md"""
    gws_path = Path(__file__).parent.parent / "backend" / "gws.md"
    try:
        with open(gws_path, "r", encoding="utf-8") as f:
            return f.read()
    except FileNotFoundError:
        print("⚠️  Warnung: gws.md nicht gefunden, verwende Grundwortschatz-Liste")
        return ""

def load_grundwortschatz_12():
    """Lädt Grundwortschatz für Klasse 1/2"""
    gws_path = Path(__file__).parent.parent / "backend" / "gws.md"
    try:
        with open(gws_path, "r", encoding="utf-8") as f:
            content = f.read()
            parts = content.split("### **Grundwortschatz für Jahrgangsstufen 3 und 4**")
            return parts[0] if len(parts) > 0 else content
    except FileNotFoundError:
        return ""

GRUNDWORTSCHATZ_FULL = load_grundwortschatz()
GRUNDWORTSCHATZ_12_TEXT = load_grundwortschatz_12()

# Konfiguration aus .env
OLLAMA_BASE_URL = os.getenv("OLLAMA_BASE_URL")
OLLAMA_API_KEY = os.getenv("OLLAMA_API_KEY")
OLLAMA_MODELS = os.getenv("OLLAMA_MODELS")
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY")
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL")
MISTRAL_MODELS = os.getenv("MISTRAL_MODELS")

# Modell-Konfigurationen aus .env aufbauen
MODEL_CONFIGS = []

# Ollama lokale Modelle hinzufügen (nur wenn konfiguriert)
if OLLAMA_BASE_URL and OLLAMA_API_KEY and OLLAMA_MODELS:
    ollama_model_list = [m.strip() for m in OLLAMA_MODELS.split(",") if m.strip()]
    for model_name in ollama_model_list:
        MODEL_CONFIGS.append({
            "name": model_name,
            "source": "ollama",
            "base_url": OLLAMA_BASE_URL,
            "api_key": OLLAMA_API_KEY
        })

# Mistral API Modelle hinzufügen (nur wenn konfiguriert)
if OPENAI_API_KEY and OPENAI_BASE_URL and MISTRAL_MODELS:
    mistral_model_list = [m.strip() for m in MISTRAL_MODELS.split(",") if m.strip()]
    for model_name in mistral_model_list:
        MODEL_CONFIGS.append({
            "name": model_name,
            "source": "mistral-api",
            "base_url": OPENAI_BASE_URL,
            "api_key": OPENAI_API_KEY
        })

# Test-Prompts (verschiedene Schwierigkeitsgrade)
TEST_PROMPTS = [
    # Klassenstufe 1-2
    {
        "name": "Klasse 1-2: Einfach - Tiere",
        "thema": "Tiere und Natur",
        "personen_tiere": "Ein kleiner Hase",
        "ort": "auf der Wiese",
        "stimmung": "fröhlich",
        "laenge": 2,
        "klassenstufe": "12"
    },
    {
        "name": "Klasse 1-2: Mittel - Freundschaft",
        "thema": "Freundschaft",
        "personen_tiere": "Ein Igel und ein Eichhörnchen",
        "ort": "im Wald",
        "stimmung": "herzlich",
        "laenge": 3,
        "klassenstufe": "12"
    },
    # Klassenstufe 3-4
    {
        "name": "Klasse 3-4: Einfach - Freundschaft",
        "thema": "Freundschaft",
        "personen_tiere": "Ein kleiner Igel",
        "ort": "im Wald",
        "stimmung": "herzlich",
        "laenge": 3,
        "klassenstufe": "34"
    },
    {
        "name": "Klasse 3-4: Mittel - Abenteuer",
        "thema": "Abenteuer",
        "personen_tiere": "Eine mutige Maus",
        "ort": "in einer alten Mühle",
        "stimmung": "spannend",
        "laenge": 5,
        "klassenstufe": "34"
    },
    {
        "name": "Klasse 3-4: Komplex - Zauber",
        "thema": "Zauber und Magie",
        "personen_tiere": "Eine junge Hexe und ihr Kater",
        "ort": "in einem verzauberten Garten",
        "stimmung": "mysteriös",
        "laenge": 3,
        "klassenstufe": "34"
    }
]

# Extrahiere Wörter aus Grundwortschatz-Text für Analyse
def extract_words_from_gws(gws_text: str) -> list:
    """Extrahiert einzelne Wörter aus dem Grundwortschatz-Text"""
    if not gws_text:
        return []
    words = re.findall(r'(?:^|\s+)-\s+([\wäöüß]+)', gws_text, re.IGNORECASE | re.MULTILINE)
    return list(set([w.lower() for w in words if w]))

GRUNDWORTSCHATZ_12_WORDS = extract_words_from_gws(GRUNDWORTSCHATZ_12_TEXT)
GRUNDWORTSCHATZ_34_WORDS = extract_words_from_gws(GRUNDWORTSCHATZ_FULL)

# Komplexe Wörter
COMPLEX_WORDS = [
    "konsequenz", "ambivalent", "rekapitulieren", "essenziell", "kontrovers",
    "paradigma", "metaphorisch", "intrinsisch", "hypothese", "analogie",
    "konzeption", "implizit", "chronologisch", "synthesieren", "abstrakt"
]

# Kreativitätsindikatoren
CREATIVE_ELEMENTS = {
    "metaphern": [r"wie ein[e]?\s+\w+", r"als ob", r"als wäre"],
    "personifikation": [r"(sonne|mond|wind|baum|blume|stern)\s+(lacht|weint|spricht|singt|tanzt|freut)"],
    "sinneswahrnehmungen": [r"(duft|geruch|roch|riecht)", r"(schmeckt|geschmack)", r"(fühl|anfühl|weich|hart|rau)"],
    "emotionale_ausdrücke": [r"(glücklich|traurig|ängstlich|mutig|fröhlich|stolz|neugierig)", r"herz\s+(klopf|schläg|hüpf)"],
    "direkte_rede": [r'[„"].*?["""]'],
}

# Altersgerechtheitskriterien
AGE_APPROPRIATE_PATTERNS = {
    "12": {
        "kurze_sätze": 8,
        "max_satzlaenge": 12,
        "min_absaetze": 2,
        "max_words_total": 200,
        "simple_words_ratio": 0.7,
    },
    "34": {
        "kurze_sätze": 15,
        "max_satzlaenge": 25,
        "min_absaetze": 3,
        "max_words_total": 500,
        "simple_words_ratio": 0.5,
    }
}


class FullModelTester:
    def __init__(self):
        self.results = []
        self.clients = {}  # Cache für API-Clients
    
    def get_client(self, base_url: str, api_key: str) -> OpenAI:
        """Holt oder erstellt einen OpenAI-Client für die gegebene URL"""
        cache_key = f"{base_url}_{api_key}"
        if cache_key not in self.clients:
            self.clients[cache_key] = OpenAI(api_key=api_key, base_url=base_url)
        return self.clients[cache_key]
    
    def create_prompt(self, test_case: dict) -> str:
        """Erstellt den Prompt für die Story-Generierung"""
        klassenstufe = test_case["klassenstufe"]
        
        if klassenstufe == "12":
            min_words = test_case["laenge"] * 60
            max_words = test_case["laenge"] * 70
            zielgruppe = "Kinder der Klassenstufen 1 & 2"
            schwierigkeit = "sehr einfach mit kurzen Sätzen und einfachen Wörtern"
            grundwortschatz = GRUNDWORTSCHATZ_12_TEXT if GRUNDWORTSCHATZ_12_TEXT else ""
        else:
            min_words = test_case["laenge"] * 80
            max_words = test_case["laenge"] * 100
            zielgruppe = "Kinder der Klassenstufen 3 & 4"
            schwierigkeit = "kindgerecht mit etwas längeren Sätzen und anspruchsvolleren Wörtern"
            grundwortschatz = GRUNDWORTSCHATZ_FULL if GRUNDWORTSCHATZ_FULL else ""
        
        prompt = f"""Du bist ein Geschichtenerzähler für {zielgruppe}.

Schreibe eine Geschichte mit folgenden Eigenschaften:
- Lesezeit: etwa {test_case['laenge']} Minuten (ca. {min_words}-{max_words} Wörter)
- Thema: {test_case['thema']}
- Personen/Tiere: {test_case['personen_tiere']}
- Ort: {test_case['ort']}
- Stimmung: {test_case['stimmung']}
- Schwierigkeitsgrad: {schwierigkeit}

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
        
        return prompt
    
    def count_words(self, text: str) -> int:
        return len(text.split())
    
    def count_paragraphs(self, text: str) -> int:
        return len([p for p in text.split('\n\n') if p.strip()])
    
    def count_dialogues(self, text: str) -> int:
        return len(re.findall(r'[„"].*?["""]', text))
    
    def analyze_grundwortschatz(self, text: str, klassenstufe: str = "34") -> dict:
        text_lower = text.lower()
        found_words = []
        gws_list = GRUNDWORTSCHATZ_12_WORDS if klassenstufe == "12" else GRUNDWORTSCHATZ_34_WORDS
        
        for word in gws_list:
            if word in text_lower:
                count = len(re.findall(r'\b' + word + r'\w*\b', text_lower))
                if count > 0:
                    found_words.append((word, count))
        
        total_occurrences = sum(count for _, count in found_words)
        unique_words = len(found_words)
        
        return {
            "unique_words": unique_words,
            "total_occurrences": total_occurrences,
            "total_gws_words": len(gws_list),
            "percentage": round((unique_words / len(gws_list)) * 100, 1),
            "top_words": sorted(found_words, key=lambda x: x[1], reverse=True)[:5]
        }
    
    def analyze_creativity(self, text: str) -> dict:
        text_lower = text.lower()
        creativity_score = {
            "metaphern": 0,
            "personifikation": 0,
            "sinneswahrnehmungen": 0,
            "emotionale_ausdrücke": 0,
            "direkte_rede": 0,
            "total_score": 0,
            "examples": []
        }
        
        for category, patterns in CREATIVE_ELEMENTS.items():
            matches = []
            for pattern in patterns:
                found = re.findall(pattern, text_lower, re.IGNORECASE)
                matches.extend(found)
            
            creativity_score[category] = len(matches)
            if matches:
                examples = matches[:2] if isinstance(matches[0], str) else [m[0] for m in matches[:2]]
                creativity_score["examples"].append({
                    "category": category,
                    "count": len(matches),
                    "samples": examples
                })
        
        creativity_score["total_score"] = sum([
            creativity_score["metaphern"] * 3,
            creativity_score["personifikation"] * 2,
            creativity_score["sinneswahrnehmungen"] * 2,
            creativity_score["emotionale_ausdrücke"],
            creativity_score["direkte_rede"]
        ])
        
        return creativity_score
    
    def analyze_age_appropriateness(self, text: str, klassenstufe: str = "34") -> dict:
        patterns = AGE_APPROPRIATE_PATTERNS.get(klassenstufe, AGE_APPROPRIATE_PATTERNS["34"])
        
        sentences = re.split(r'[.!?]+', text)
        sentences = [s.strip() for s in sentences if s.strip()]
        
        sentence_lengths = [len(s.split()) for s in sentences]
        avg_sentence_length = sum(sentence_lengths) / len(sentence_lengths) if sentence_lengths else 0
        long_sentences = len([l for l in sentence_lengths if l > patterns["max_satzlaenge"]])
        
        text_lower = text.lower()
        complex_words_found = [word for word in COMPLEX_WORDS if word in text_lower]
        
        paragraph_count = self.count_paragraphs(text)
        
        words = re.findall(r'\b\w+\b', text_lower)
        unique_words = set(words)
        ttr = len(unique_words) / len(words) if words else 0
        total_words = len(words)
        
        gws_list = GRUNDWORTSCHATZ_12_WORDS if klassenstufe == "12" else GRUNDWORTSCHATZ_34_WORDS
        gws_count = sum(1 for word in words if any(gws in word for gws in gws_list))
        gws_ratio = gws_count / total_words if total_words else 0
        
        score = 100
        issues = []
        
        if avg_sentence_length > patterns["kurze_sätze"]:
            penalty = 15 if klassenstufe == "12" else 10
            score -= penalty
            issues.append(f"Sätze zu lang (Ø {avg_sentence_length:.1f} Wörter, Ziel: <{patterns['kurze_sätze']})")
        
        if long_sentences > len(sentences) * 0.3:
            penalty = 20 if klassenstufe == "12" else 15
            score -= penalty
            issues.append(f"{long_sentences} Sätze über {patterns['max_satzlaenge']} Wörter")
        
        if complex_words_found:
            penalty = 10 if klassenstufe == "12" else 5
            score -= len(complex_words_found) * penalty
            issues.append(f"{len(complex_words_found)} zu komplexe Wörter")
        
        if paragraph_count < patterns["min_absaetze"]:
            score -= 10
            issues.append(f"Zu wenig Absätze ({paragraph_count}, Ziel: >={patterns['min_absaetze']})")
        
        if total_words > patterns["max_words_total"]:
            score -= 10
            issues.append(f"Geschichte zu lang ({total_words} Wörter, Ziel: <{patterns['max_words_total']})")
        
        if klassenstufe == "12":
            if gws_ratio < patterns["simple_words_ratio"]:
                score -= 15
                issues.append(f"Zu wenig Grundwortschatz ({gws_ratio:.1%}, Ziel: >{patterns['simple_words_ratio']:.0%})")
            if ttr > 0.7:
                score -= 10
                issues.append(f"Zu viel Wortvielfalt (TTR: {ttr:.2f}, mehr Wiederholungen wären besser)")
        else:
            if ttr < 0.4:
                score -= 5
                issues.append(f"Geringe Wortvielfalt (TTR: {ttr:.2f})")
        
        return {
            "score": max(0, score),
            "klassenstufe": klassenstufe,
            "avg_sentence_length": round(avg_sentence_length, 1),
            "long_sentences": long_sentences,
            "total_sentences": len(sentences),
            "total_words": total_words,
            "complex_words": complex_words_found,
            "paragraph_count": paragraph_count,
            "type_token_ratio": round(ttr, 2),
            "grundwortschatz_ratio": round(gws_ratio, 2),
            "issues": issues,
            "recommendation": "Sehr gut" if score >= 90 else "Gut" if score >= 75 else "Verbesserungswürdig" if score >= 50 else "Ungeeignet"
        }
    
    def test_model(self, model_config: dict, test_case: dict) -> dict:
        """Testet ein Modell mit einem Test-Case"""
        model_name = model_config["name"]
        source = model_config["source"]
        print(f"  📝 Teste: {test_case['name']}")
        
        prompt = self.create_prompt(test_case)
        client = self.get_client(model_config["base_url"], model_config["api_key"])
        
        start_time = time.time()
        try:
            estimated_tokens = int(test_case["laenge"] * 100 * 1.3) + 200
            
            response = client.chat.completions.create(
                model=model_name,
                messages=[
                    {"role": "system", "content": "Du bist ein kreativer Geschichtenerzähler für Grundschulkinder."},
                    {"role": "user", "content": prompt}
                ],
                temperature=0.8,
                max_tokens=estimated_tokens
            )
            generation_time = time.time() - start_time
            
            content = response.choices[0].message.content or ""
            
            title = "Ohne Titel"
            story = content
            
            if "TITEL:" in content:
                parts = content.split("TITEL:", 1)
                if len(parts) > 1:
                    rest = parts[1].strip()
                    title_end = rest.find("\n")
                    if title_end > 0:
                        title = rest[:title_end].strip()
                        story = rest[title_end+1:].strip()
            
            word_count = self.count_words(story)
            paragraph_count = self.count_paragraphs(story)
            dialogue_count = self.count_dialogues(story)
            gws_analysis = self.analyze_grundwortschatz(story, test_case["klassenstufe"])
            creativity_analysis = self.analyze_creativity(story)
            age_analysis = self.analyze_age_appropriateness(story, test_case["klassenstufe"])
            
            tokens_used = response.usage.total_tokens if hasattr(response, 'usage') and response.usage else 0
            
            result = {
                "test_case": test_case["name"],
                "success": True,
                "source": source,
                "generation_time": round(generation_time, 2),
                "title": title,
                "word_count": word_count,
                "paragraph_count": paragraph_count,
                "dialogue_count": dialogue_count,
                "grundwortschatz": gws_analysis,
                "creativity": creativity_analysis,
                "age_appropriateness": age_analysis,
                "tokens_used": tokens_used,
                "story_preview": story[:200] + "..." if len(story) > 200 else story
            }
            
            print(f"    ✅ {generation_time:.1f}s | {word_count} Wörter | Kreativ: {creativity_analysis['total_score']} | Altersgerecht: {age_analysis['score']}/100")
            
        except Exception as e:
            result = {
                "test_case": test_case["name"],
                "success": False,
                "source": source,
                "error": str(e),
                "generation_time": time.time() - start_time
            }
            print(f"    ❌ Fehler: {str(e)}")
        
        return result
    
    def unload_ollama_model(self, model: str):
        """Entlädt ein Ollama-Modell aus dem Speicher"""
        if not OLLAMA_BASE_URL:
            return
        try:
            base_url = OLLAMA_BASE_URL.replace("/v1", "")
            response = requests.post(
                f"{base_url}/api/generate",
                json={"model": model, "keep_alive": 0}
            )
            if response.status_code == 200:
                print(f"  📤 Modell {model} entladen")
        except Exception as e:
            print(f"  ⚠️  Fehler beim Entladen von {model}: {str(e)}")
    
    def test_all_models(self):
        """Testet alle konfigurierten Modelle"""
        print("🧪 Starte vollständigen Modell-Vergleichstest\n")
        print(f"📋 {len(MODEL_CONFIGS)} Modelle × {len(TEST_PROMPTS)} Test-Cases = {len(MODEL_CONFIGS) * len(TEST_PROMPTS)} Tests\n")
        
        ollama_count = sum(1 for m in MODEL_CONFIGS if m["source"] == "ollama")
        mistral_count = sum(1 for m in MODEL_CONFIGS if m["source"] == "mistral-api")
        print(f"🔧 Ollama lokal: {ollama_count} Modelle")
        print(f"🌐 Mistral API: {mistral_count} Modelle\n")
        
        for model_config in MODEL_CONFIGS:
            model_name = model_config["name"]
            source = model_config["source"]
            source_label = "Ollama" if source == "ollama" else "Mistral API"
            
            print(f"\n{'='*60}")
            print(f"🤖 Modell: {model_name} ({source_label})")
            print(f"{'='*60}")
            
            model_results = {
                "model": model_name,
                "source": source,
                "base_url": model_config["base_url"],
                "timestamp": datetime.now().isoformat(),
                "tests": []
            }
            
            for test_case in TEST_PROMPTS:
                result = self.test_model(model_config, test_case)
                model_results["tests"].append(result)
                time.sleep(1)
            
            self.results.append(model_results)
            
            # Entlade nur Ollama-Modelle
            if source == "ollama":
                self.unload_ollama_model(model_name)
            print()
        
        print(f"\n{'='*60}")
        print("✅ Alle Tests abgeschlossen!")
        print(f"{'='*60}\n")
    
    def generate_report(self) -> str:
        """Generiert einen vollständigen Vergleichsbericht"""
        report = ["# 📊 Vollständiger Modell-Vergleichsbericht - Kindergeschichten\n"]
        report.append(f"**Datum:** {datetime.now().strftime('%d.%m.%Y %H:%M')}\n")
        report.append(f"**Getestete Modelle:** {len(MODEL_CONFIGS)}\n")
        report.append(f"**Test-Cases:** {len(TEST_PROMPTS)}\n\n")
        
        # Übersichtstabelle
        report.append("## 📈 Gesamtübersicht\n")
        report.append("| Modell | Quelle | Ø Zeit (s) | Ø Wörter | Kreativität | Altersgerecht | Erfolg |\n")
        report.append("|--------|--------|-----------|----------|-------------|---------------|--------|\n")
        
        for model_result in self.results:
            model = model_result["model"]
            source = "🔧 Ollama" if model_result["source"] == "ollama" else "🌐 Mistral API"
            tests = model_result["tests"]
            successful_tests = [t for t in tests if t.get("success")]
            
            if successful_tests:
                avg_time = sum(t["generation_time"] for t in successful_tests) / len(successful_tests)
                avg_words = sum(t["word_count"] for t in successful_tests) / len(successful_tests)
                avg_creativity = sum(t["creativity"]["total_score"] for t in successful_tests) / len(successful_tests)
                avg_age_score = sum(t["age_appropriateness"]["score"] for t in successful_tests) / len(successful_tests)
                success_rate = f"{len(successful_tests)}/{len(tests)}"
                
                report.append(f"| {model} | {source} | {avg_time:.1f} | {avg_words:.0f} | {avg_creativity:.0f} | {avg_age_score:.0f}/100 | {success_rate} |\n")
        
        # Vergleich nach Quelle
        report.append("\n## 🔍 Vergleich nach Quelle\n")
        
        ollama_results = [r for r in self.results if r["source"] == "ollama"]
        mistral_results = [r for r in self.results if r["source"] == "mistral-api"]
        
        if ollama_results:
            report.append("\n### 🔧 Ollama Modelle\n")
            for model_result in ollama_results:
                tests = [t for t in model_result["tests"] if t.get("success")]
                if tests:
                    avg_time = sum(t["generation_time"] for t in tests) / len(tests)
                    avg_creativity = sum(t["creativity"]["total_score"] for t in tests) / len(tests)
                    avg_age = sum(t["age_appropriateness"]["score"] for t in tests) / len(tests)
                    report.append(f"- **{model_result['model']}**: Zeit {avg_time:.1f}s | Kreativ {avg_creativity:.0f} | Altersgerecht {avg_age:.0f}/100\n")
        
        if mistral_results:
            report.append("\n### 🌐 Mistral API Modelle\n")
            for model_result in mistral_results:
                tests = [t for t in model_result["tests"] if t.get("success")]
                if tests:
                    avg_time = sum(t["generation_time"] for t in tests) / len(tests)
                    avg_creativity = sum(t["creativity"]["total_score"] for t in tests) / len(tests)
                    avg_age = sum(t["age_appropriateness"]["score"] for t in tests) / len(tests)
                    report.append(f"- **{model_result['model']}**: Zeit {avg_time:.1f}s | Kreativ {avg_creativity:.0f} | Altersgerecht {avg_age:.0f}/100\n")
        
        # Detaillierte Ergebnisse
        report.append("\n## 📝 Detaillierte Ergebnisse\n")
        
        for model_result in self.results:
            model = model_result["model"]
            source = "🔧 Ollama" if model_result["source"] == "ollama" else "🌐 Mistral API"
            report.append(f"\n### {source} - {model}\n")
            
            for test in model_result["tests"]:
                if test.get("success"):
                    report.append(f"\n#### {test['test_case']}\n")
                    report.append(f"- **Zeit:** {test['generation_time']:.1f}s\n")
                    report.append(f"- **Titel:** {test['title']}\n")
                    report.append(f"- **Wörter:** {test['word_count']}\n")
                    report.append(f"- **Absätze:** {test['paragraph_count']}\n")
                    report.append(f"- **Dialoge:** {test['dialogue_count']}\n")
                    
                    gws = test['grundwortschatz']
                    report.append(f"- **Grundwortschatz:** {gws['unique_words']}/{gws['total_gws_words']} Wörter ({gws['percentage']}%)\n")
                    
                    creativity = test['creativity']
                    report.append(f"- **Kreativitäts-Score:** {creativity['total_score']}\n")
                    
                    age = test['age_appropriateness']
                    klassenstufe_name = "Klasse 1-2" if age.get('klassenstufe') == "12" else "Klasse 3-4"
                    report.append(f"- **Altersangemessenheit ({klassenstufe_name}):** {age['score']}/100 ({age['recommendation']})\n")
                    
                    report.append(f"- **Tokens:** {test['tokens_used']}\n")
                    report.append(f"\n**Auszug:**\n> {test['story_preview']}\n")
                else:
                    report.append(f"\n#### ❌ {test['test_case']}\n")
                    report.append(f"- **Fehler:** {test.get('error', 'Unbekannter Fehler')}\n")
        
        # Empfehlungen
        report.append("\n## 🏆 Empfehlungen\n")
        
        if self.results:
            fastest = min(self.results, key=lambda x: sum(t["generation_time"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
            report.append(f"- **⚡ Schnellstes Modell:** {fastest['model']} ({fastest['source']})\n")
            
            most_creative = max(self.results, key=lambda x: sum(t["creativity"]["total_score"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
            report.append(f"- **🎨 Kreativstes Modell:** {most_creative['model']} ({most_creative['source']})\n")
            
            best_age = max(self.results, key=lambda x: sum(t["age_appropriateness"]["score"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
            report.append(f"- **👶 Am besten für Altersgruppe:** {best_age['model']} ({best_age['source']})\n")
            
            best_gws = max(self.results, key=lambda x: sum(t["grundwortschatz"]["unique_words"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
            report.append(f"- **📚 Bester Grundwortschatz:** {best_gws['model']} ({best_gws['source']})\n")
            
            # Gesamtbewertung
            report.append("\n### 🎯 Gesamtbewertung (Gewichteter Score)\n")
            report.append("*Berechnung: Kreativität × 2 + Altersgerecht × 3 + GWS-Wörter × 1 - Zeit/50*\n\n")
            
            overall_scores = []
            for model_result in self.results:
                tests = [t for t in model_result["tests"] if t.get("success")]
                if tests:
                    avg_creativity = sum(t["creativity"]["total_score"] for t in tests) / len(tests)
                    avg_age = sum(t["age_appropriateness"]["score"] for t in tests) / len(tests)
                    avg_gws = sum(t["grundwortschatz"]["unique_words"] for t in tests) / len(tests)
                    avg_time = sum(t["generation_time"] for t in tests) / len(tests)
                    
                    weighted_score = (avg_creativity * 2) + (avg_age * 3) + (avg_gws * 1) - (avg_time / 50)
                    source_icon = "🔧" if model_result["source"] == "ollama" else "🌐"
                    overall_scores.append((model_result["model"], weighted_score, source_icon))
            
            overall_scores.sort(key=lambda x: x[1], reverse=True)
            
            for i, (model, score, source_icon) in enumerate(overall_scores, 1):
                medal = "🥇" if i == 1 else "🥈" if i == 2 else "🥉" if i == 3 else f"{i}."
                report.append(f"{medal} **{model}** {source_icon} - Score: {score:.1f}\n")
        
        return "".join(report)
    
    def save_results(self, output_dir: str = "test_results"):
        """Speichert Ergebnisse als JSON und Markdown"""
        output_path = Path(output_dir)
        output_path.mkdir(exist_ok=True)
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        json_file = output_path / f"full_results_{timestamp}.json"
        with open(json_file, "w", encoding="utf-8") as f:
            json.dump(self.results, f, indent=2, ensure_ascii=False)
        print(f"💾 JSON gespeichert: {json_file}")
        
        report = self.generate_report()
        md_file = output_path / f"full_report_{timestamp}.md"
        with open(md_file, "w", encoding="utf-8") as f:
            f.write(report)
        print(f"📄 Report gespeichert: {md_file}")
        
        latest_md = output_path / "latest_full_report.md"
        with open(latest_md, "w", encoding="utf-8") as f:
            f.write(report)
        print(f"📄 Latest Report: {latest_md}")


def main():
    """Hauptfunktion"""
    print("\n" + "="*60)
    print("🧪 mAIrchen - Vollständiger Modell-Vergleichstest")
    print("="*60 + "\n")
    
    # Prüfe Konfiguration
    if not MODEL_CONFIGS:
        print("❌ Keine Modelle konfiguriert!")
        print("   Bitte prüfe die .env Datei im tools/ Ordner.")
        print("   Stelle sicher, dass mindestens eine Provider-Konfiguration aktiv ist:")
        print("   - Ollama: OLLAMA_BASE_URL, OLLAMA_API_KEY, OLLAMA_MODELS")
        print("   - Mistral API: OPENAI_API_KEY, OPENAI_BASE_URL, MISTRAL_MODELS")
        sys.exit(1)
    
    # Zeige Konfiguration
    if OLLAMA_BASE_URL and OLLAMA_MODELS:
        print(f"🔧 Ollama Base URL: {OLLAMA_BASE_URL}")
        print(f"   Modelle: {OLLAMA_MODELS}")
    else:
        print("⚠️  Ollama nicht konfiguriert (auskommentiert oder fehlend)")
    
    if OPENAI_API_KEY and OPENAI_BASE_URL and MISTRAL_MODELS:
        print(f"🌐 Mistral API: {OPENAI_BASE_URL}")
        print(f"   Modelle: {MISTRAL_MODELS}")
    else:
        print("⚠️  Mistral API nicht konfiguriert (auskommentiert oder fehlend)")
    print()
    
    tester = FullModelTester()
    tester.test_all_models()
    tester.save_results()
    
    print("\n" + "="*60)
    print("📊 ZUSAMMENFASSUNG")
    print("="*60 + "\n")
    print(tester.generate_report())


if __name__ == "__main__":
    main()
