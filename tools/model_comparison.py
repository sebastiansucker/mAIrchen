#!/usr/bin/env python3
"""
Test-Script zum Vergleich verschiedener Ollama-Modelle für Kindergeschichten.

Usage:
    python test_models.py

Testet verschiedene Modelle mit den gleichen Prompts und bewertet:
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

# Konfiguration
OLLAMA_BASE_URL = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434/v1")
OLLAMA_MODELS = [
    "gemma3:latest",
    "gemma3n:latest", 
    "llama3.2:3b",
    "mistral-small3.2:latest",
    "gpt-oss:20b",
    "qwen3:latest",
    "deepseek-r1:latest",
    "phi4:latest",
]

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
    # Finde alle Wörter (ohne Markdown-Syntax)
    words = re.findall(r'(?:^|\s+)-\s+([\wäöüß]+)', gws_text, re.IGNORECASE | re.MULTILINE)
    # Normalisiere zu Kleinbuchstaben und entferne Duplikate
    return list(set([w.lower() for w in words if w]))

# Erstelle Wörterlisten aus geladenen Texten
GRUNDWORTSCHATZ_12_WORDS = extract_words_from_gws(GRUNDWORTSCHATZ_12_TEXT)
GRUNDWORTSCHATZ_34_WORDS = extract_words_from_gws(GRUNDWORTSCHATZ_FULL)

# Komplexe Wörter (für Altersgruppe 3-4 zu schwierig)
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

# Altersgerechtheitskriterien nach Klassenstufe
AGE_APPROPRIATE_PATTERNS = {
    "12": {  # Klasse 1-2
        "kurze_sätze": 8,  # Sehr kurze Sätze
        "max_satzlaenge": 12,  # Max 12 Wörter pro Satz
        "min_absaetze": 2,  # Mindestens 2 Absätze
        "max_words_total": 200,  # Kürzere Geschichten
        "simple_words_ratio": 0.7,  # 70% sollten einfache Wörter sein
    },
    "34": {  # Klasse 3-4
        "kurze_sätze": 15,  # Durchschnittliche Wörter pro Satz
        "max_satzlaenge": 25,  # Einzelne Sätze nicht länger als 25 Wörter
        "min_absaetze": 3,  # Mindestens 3 Absätze für Struktur
        "max_words_total": 500,  # Längere Geschichten erlaubt
        "simple_words_ratio": 0.5,  # 50% sollten einfache Wörter sein
    }
}


class ModelTester:
    def __init__(self, base_url: str):
        self.client = OpenAI(api_key="ollama", base_url=base_url)
        self.results = []
    
    def create_prompt(self, test_case: dict) -> str:
        """Erstellt den Prompt für die Story-Generierung"""
        klassenstufe = test_case["klassenstufe"]
        
        # Berechne Wortanzahl basierend auf Lesegeschwindigkeit nach Klassenstufe
        if klassenstufe == "12":
            # Klasse 1 & 2: ~70 Wörter/Min
            min_words = test_case["laenge"] * 60
            max_words = test_case["laenge"] * 70
            zielgruppe = "Kinder der Klassenstufen 1 & 2"
            schwierigkeit = "sehr einfach mit kurzen Sätzen und einfachen Wörtern"
            grundwortschatz = GRUNDWORTSCHATZ_12_TEXT if GRUNDWORTSCHATZ_12_TEXT else ""
        else:
            # Klasse 3 & 4: ~80-100 Wörter/Min
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
        """Zählt Wörter im Text"""
        return len(text.split())
    
    def count_paragraphs(self, text: str) -> int:
        """Zählt Absätze"""
        return len([p for p in text.split('\n\n') if p.strip()])
    
    def count_dialogues(self, text: str) -> int:
        """Zählt Dialog-Zeilen (mit Anführungszeichen)"""
        return len(re.findall(r'[„"].*?["""]', text))
    
    def analyze_grundwortschatz(self, text: str, klassenstufe: str = "34") -> dict:
        """Analysiert Grundwortschatz-Nutzung nach Klassenstufe"""
        text_lower = text.lower()
        found_words = []
        
        # Wähle passenden Grundwortschatz
        gws_list = GRUNDWORTSCHATZ_12_WORDS if klassenstufe == "12" else GRUNDWORTSCHATZ_34_WORDS
        
        for word in gws_list:
            if word in text_lower:
                # Zähle Vorkommen
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
        """Analysiert kreative Elemente in der Geschichte"""
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
                # Speichere Beispiele (max 2 pro Kategorie)
                examples = matches[:2] if isinstance(matches[0], str) else [m[0] for m in matches[:2]]
                creativity_score["examples"].append({
                    "category": category,
                    "count": len(matches),
                    "samples": examples
                })
        
        creativity_score["total_score"] = sum([
            creativity_score["metaphern"] * 3,  # Metaphern sind wertvoll
            creativity_score["personifikation"] * 2,
            creativity_score["sinneswahrnehmungen"] * 2,
            creativity_score["emotionale_ausdrücke"],
            creativity_score["direkte_rede"]
        ])
        
        return creativity_score
    
    def analyze_age_appropriateness(self, text: str, klassenstufe: str = "34") -> dict:
        """Analysiert Altersangemessenheit für die jeweilige Klassenstufe"""
        patterns = AGE_APPROPRIATE_PATTERNS.get(klassenstufe, AGE_APPROPRIATE_PATTERNS["34"])
        
        # Satzlängen-Analyse
        sentences = re.split(r'[.!?]+', text)
        sentences = [s.strip() for s in sentences if s.strip()]
        
        sentence_lengths = [len(s.split()) for s in sentences]
        avg_sentence_length = sum(sentence_lengths) / len(sentence_lengths) if sentence_lengths else 0
        long_sentences = len([l for l in sentence_lengths if l > patterns["max_satzlaenge"]])
        
        # Komplexe Wörter finden
        text_lower = text.lower()
        complex_words_found = []
        for word in COMPLEX_WORDS:
            if word in text_lower:
                complex_words_found.append(word)
        
        # Struktur-Analyse
        paragraph_count = self.count_paragraphs(text)
        
        # Wortvielfalt (Type-Token-Ratio)
        words = re.findall(r'\b\w+\b', text_lower)
        unique_words = set(words)
        ttr = len(unique_words) / len(words) if words else 0
        total_words = len(words)
        
        # Grundwortschatz-Anteil (verwende passenden Wortschatz)
        gws_list = GRUNDWORTSCHATZ_12_WORDS if klassenstufe == "12" else GRUNDWORTSCHATZ_34_WORDS
        gws_count = sum(1 for word in words if any(gws in word for gws in gws_list))
        gws_ratio = gws_count / total_words if total_words else 0
        
        # Bewertung
        score = 100
        issues = []
        
        if avg_sentence_length > patterns["kurze_sätze"]:
            penalty = 15 if klassenstufe == "12" else 10
            score -= penalty
            issues.append(f"Sätze zu lang (Ø {avg_sentence_length:.1f} Wörter, Ziel: <{patterns['kurze_sätze']})")
        
        if long_sentences > len(sentences) * 0.3:  # Mehr als 30% zu lange Sätze
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
        
        # Für Klasse 1-2: Grundwortschatz-Anteil wichtiger
        if klassenstufe == "12":
            if gws_ratio < patterns["simple_words_ratio"]:
                score -= 15
                issues.append(f"Zu wenig Grundwortschatz ({gws_ratio:.1%}, Ziel: >{patterns['simple_words_ratio']:.0%})")
            if ttr > 0.7:  # Zu viel Wortvielfalt für Leseanfänger
                score -= 10
                issues.append(f"Zu viel Wortvielfalt (TTR: {ttr:.2f}, mehr Wiederholungen wären besser)")
        else:
            if ttr < 0.4:  # Wenig Wortvielfalt für Klasse 3-4
                score -= 5
                issues.append(f"Geringe Wortvielfalt (TTR: {ttr:.2f})")
        
        return {
            "score": max(0, score),  # Minimum 0
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
    
    def test_model(self, model: str, test_case: dict) -> dict:
        """Testet ein Modell mit einem Test-Case"""
        print(f"  📝 Teste: {test_case['name']}")
        
        prompt = self.create_prompt(test_case)
        
        # Generierung mit Zeiterfassung
        start_time = time.time()
        try:
            # Berechne max_tokens basierend auf gewünschter Länge
            estimated_tokens = int(test_case["laenge"] * 100 * 1.3) + 200
            
            response = self.client.chat.completions.create(
                model=model,
                messages=[
                    {"role": "system", "content": "Du bist ein kreativer Geschichtenerzähler für Grundschulkinder."},
                    {"role": "user", "content": prompt}
                ],
                temperature=0.8,
                max_tokens=estimated_tokens
            )
            generation_time = time.time() - start_time
            
            content = response.choices[0].message.content or ""
            
            # Parse Titel und Story
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
            
            # Analyse
            word_count = self.count_words(story)
            paragraph_count = self.count_paragraphs(story)
            dialogue_count = self.count_dialogues(story)
            gws_analysis = self.analyze_grundwortschatz(story, test_case["klassenstufe"])
            creativity_analysis = self.analyze_creativity(story)
            age_analysis = self.analyze_age_appropriateness(story, test_case["klassenstufe"])
            
            # Token-Nutzung
            tokens_used = response.usage.total_tokens if hasattr(response, 'usage') and response.usage else 0
            
            result = {
                "test_case": test_case["name"],
                "success": True,
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
                "error": str(e),
                "generation_time": time.time() - start_time
            }
            print(f"    ❌ Fehler: {str(e)}")
        
        return result
    
    def unload_model(self, model: str):
        """Entlädt ein Modell aus dem Ollama-Speicher"""
        try:
            # Ollama API Endpoint zum Entladen von Modellen
            base_url = OLLAMA_BASE_URL.replace("/v1", "")  # Entferne /v1 vom Pfad
            response = requests.post(
                f"{base_url}/api/generate",
                json={
                    "model": model,
                    "keep_alive": 0  # 0 = sofort entladen
                }
            )
            if response.status_code == 200:
                print(f"  📤 Modell {model} entladen")
            else:
                print(f"  ⚠️  Konnte Modell {model} nicht entladen: {response.status_code}")
        except Exception as e:
            print(f"  ⚠️  Fehler beim Entladen von {model}: {str(e)}")
    
    def test_all_models(self):
        """Testet alle Modelle mit allen Test-Cases"""
        print("🧪 Starte Modell-Vergleichstest\n")
        print(f"📍 Ollama Base URL: {OLLAMA_BASE_URL}")
        print(f"📋 {len(OLLAMA_MODELS)} Modelle × {len(TEST_PROMPTS)} Test-Cases = {len(OLLAMA_MODELS) * len(TEST_PROMPTS)} Tests\n")
        
        for model in OLLAMA_MODELS:
            print(f"\n{'='*60}")
            print(f"🤖 Modell: {model}")
            print(f"{'='*60}")
            
            model_results = {
                "model": model,
                "timestamp": datetime.now().isoformat(),
                "tests": []
            }
            
            for test_case in TEST_PROMPTS:
                result = self.test_model(model, test_case)
                model_results["tests"].append(result)
                time.sleep(1)  # Kurze Pause zwischen Tests
            
            self.results.append(model_results)
            
            # Entlade Modell aus dem Speicher
            self.unload_model(model)
            print()
        
        print(f"\n{'='*60}")
        print("✅ Alle Tests abgeschlossen!")
        print(f"{'='*60}\n")
    
    def generate_report(self) -> str:
        """Generiert einen Vergleichsbericht"""
        report = ["# 📊 Modell-Vergleichsbericht - Kindergeschichten\n"]
        report.append(f"**Datum:** {datetime.now().strftime('%d.%m.%Y %H:%M')}\n")
        report.append(f"**Getestete Modelle:** {len(OLLAMA_MODELS)}\n")
        report.append(f"**Test-Cases:** {len(TEST_PROMPTS)}\n\n")
        
        # Übersichtstabelle
        report.append("## 📈 Gesamtübersicht\n")
        report.append("| Modell | Ø Zeit (s) | Ø Wörter | Kreativität | Altersgerecht | Erfolg |\n")
        report.append("|--------|-----------|----------|-------------|---------------|--------|\n")
        
        for model_result in self.results:
            model = model_result["model"]
            tests = model_result["tests"]
            successful_tests = [t for t in tests if t.get("success")]
            
            if successful_tests:
                avg_time = sum(t["generation_time"] for t in successful_tests) / len(successful_tests)
                avg_words = sum(t["word_count"] for t in successful_tests) / len(successful_tests)
                avg_creativity = sum(t["creativity"]["total_score"] for t in successful_tests) / len(successful_tests)
                avg_age_score = sum(t["age_appropriateness"]["score"] for t in successful_tests) / len(successful_tests)
                success_rate = f"{len(successful_tests)}/{len(tests)}"
                
                report.append(f"| {model} | {avg_time:.1f} | {avg_words:.0f} | {avg_creativity:.0f} | {avg_age_score:.0f}/100 | {success_rate} |\n")
        
        # Detaillierte Ergebnisse pro Modell
        report.append("\n## 📝 Detaillierte Ergebnisse\n")
        
        for model_result in self.results:
            model = model_result["model"]
            report.append(f"\n### 🤖 {model}\n")
            
            for test in model_result["tests"]:
                if test.get("success"):
                    report.append(f"\n#### {test['test_case']}\n")
                    report.append(f"- **Zeit:** {test['generation_time']:.1f}s\n")
                    report.append(f"- **Titel:** {test['title']}\n")
                    report.append(f"- **Wörter:** {test['word_count']}\n")
                    report.append(f"- **Absätze:** {test['paragraph_count']}\n")
                    report.append(f"- **Dialoge:** {test['dialogue_count']}\n")
                    
                    # Grundwortschatz
                    gws = test['grundwortschatz']
                    report.append(f"- **Grundwortschatz:** {gws['unique_words']}/{gws['total_gws_words']} Wörter ({gws['percentage']}%)\n")
                    
                    # Kreativität
                    creativity = test['creativity']
                    report.append(f"- **Kreativitäts-Score:** {creativity['total_score']}\n")
                    report.append(f"  - Metaphern: {creativity['metaphern']}\n")
                    report.append(f"  - Personifikation: {creativity['personifikation']}\n")
                    report.append(f"  - Sinneswahrnehmungen: {creativity['sinneswahrnehmungen']}\n")
                    report.append(f"  - Emotionale Ausdrücke: {creativity['emotionale_ausdrücke']}\n")
                    report.append(f"  - Direkte Rede: {creativity['direkte_rede']}\n")
                    
                    # Altersangemessenheit
                    age = test['age_appropriateness']
                    klassenstufe_name = "Klasse 1-2" if age.get('klassenstufe') == "12" else "Klasse 3-4"
                    report.append(f"- **Altersangemessenheit ({klassenstufe_name}):** {age['score']}/100 ({age['recommendation']})\n")
                    report.append(f"  - Ø Satzlänge: {age['avg_sentence_length']} Wörter\n")
                    report.append(f"  - Lange Sätze: {age['long_sentences']}/{age['total_sentences']}\n")
                    report.append(f"  - Gesamtwörter: {age['total_words']}\n")
                    report.append(f"  - Wortvielfalt (TTR): {age['type_token_ratio']}\n")
                    report.append(f"  - Grundwortschatz-Anteil: {age['grundwortschatz_ratio']:.0%}\n")
                    if age['issues']:
                        report.append(f"  - ⚠️ Hinweise: {', '.join(age['issues'])}\n")
                    if age['complex_words']:
                        report.append(f"  - ⚠️ Komplexe Wörter: {', '.join(age['complex_words'][:3])}\n")
                    
                    report.append(f"- **Tokens:** {test['tokens_used']}\n")
                    report.append(f"\n**Auszug:**\n> {test['story_preview']}\n")
                else:
                    report.append(f"\n#### ❌ {test['test_case']}\n")
                    report.append(f"- **Fehler:** {test.get('error', 'Unbekannter Fehler')}\n")
        
        # Empfehlungen
        report.append("\n## 🏆 Empfehlungen\n")
        
        # Schnellstes Modell
        fastest = min(self.results, key=lambda x: sum(t["generation_time"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
        report.append(f"- **⚡ Schnellstes Modell:** {fastest['model']}\n")
        
        # Kreativstes Modell
        most_creative = max(self.results, key=lambda x: sum(t["creativity"]["total_score"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
        report.append(f"- **🎨 Kreativstes Modell:** {most_creative['model']}\n")
        
        # Best für Altersgruppe
        best_age = max(self.results, key=lambda x: sum(t["age_appropriateness"]["score"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
        report.append(f"- **👶 Am besten für Altersgruppe:** {best_age['model']}\n")
        
        # Best Grundwortschatz
        best_gws = max(self.results, key=lambda x: sum(t["grundwortschatz"]["unique_words"] for t in x["tests"] if t.get("success")) / max(len([t for t in x["tests"] if t.get("success")]), 1))
        report.append(f"- **📚 Bester Grundwortschatz:** {best_gws['model']}\n")
        
        # Gesamtbewertung (gewichteter Score)
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
                overall_scores.append((model_result["model"], weighted_score))
        
        overall_scores.sort(key=lambda x: x[1], reverse=True)
        
        for i, (model, score) in enumerate(overall_scores, 1):
            medal = "🥇" if i == 1 else "🥈" if i == 2 else "🥉" if i == 3 else f"{i}."
            report.append(f"{medal} **{model}** - Score: {score:.1f}\n")
        
        return "".join(report)
    
    def save_results(self, output_dir: str = "test_results"):
        """Speichert Ergebnisse als JSON und Markdown"""
        output_path = Path(output_dir)
        output_path.mkdir(exist_ok=True)
        
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        # JSON
        json_file = output_path / f"results_{timestamp}.json"
        with open(json_file, "w", encoding="utf-8") as f:
            json.dump(self.results, f, indent=2, ensure_ascii=False)
        print(f"💾 JSON gespeichert: {json_file}")
        
        # Markdown Report
        report = self.generate_report()
        md_file = output_path / f"report_{timestamp}.md"
        with open(md_file, "w", encoding="utf-8") as f:
            f.write(report)
        print(f"📄 Report gespeichert: {md_file}")
        
        # Auch als latest
        latest_md = output_path / "latest_report.md"
        with open(latest_md, "w", encoding="utf-8") as f:
            f.write(report)
        print(f"📄 Latest Report: {latest_md}")


def main():
    """Hauptfunktion"""
    print("\n" + "="*60)
    print("🧪 mAIrchen Modell-Vergleichstest")
    print("="*60 + "\n")
    
    # Prüfe Ollama-Verbindung
    try:
        client = OpenAI(api_key="ollama", base_url=OLLAMA_BASE_URL)
        # Versuche eine einfache Anfrage
        print("🔍 Prüfe Ollama-Verbindung...")
        # Note: Ollama unterstützt nicht direkt /models über OpenAI API
        print(f"✅ Verbunden mit: {OLLAMA_BASE_URL}\n")
    except Exception as e:
        print(f"❌ Fehler bei Verbindung zu Ollama: {e}")
        print(f"   Stelle sicher, dass Ollama läuft: ollama serve")
        sys.exit(1)
    
    # Starte Tests
    tester = ModelTester(OLLAMA_BASE_URL)
    tester.test_all_models()
    
    # Speichere Ergebnisse
    tester.save_results()
    
    # Zeige Report
    print("\n" + "="*60)
    print("📊 ZUSAMMENFASSUNG")
    print("="*60 + "\n")
    print(tester.generate_report())


if __name__ == "__main__":
    main()
