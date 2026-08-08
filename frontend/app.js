// API Base URL - verwende relative URLs damit es von jedem Gerät funktioniert
const API_URL = window.location.origin;

// DOM Elemente
const inputForm = document.getElementById('input-form');
const storyDisplay = document.getElementById('story-display');
const loading = document.getElementById('loading');

const themaInput = document.getElementById('thema');
const personenInput = document.getElementById('personen');
const ortInput = document.getElementById('ort');
const stimmungInput = document.getElementById('stimmung');
const stilInput = document.getElementById('stil');
const lengthButtons = document.querySelectorAll('.length-btn');
const gradeButtons = document.querySelectorAll('.grade-btn');
const moodChips = document.querySelectorAll('.mood-chip');
let selectedLength = 10; // Standard: 10 Minuten
let selectedGrade = '34'; // Standard: 3/4 Klasse

// Length Button Event Listeners
lengthButtons.forEach(btn => {
    btn.addEventListener('click', () => {
        lengthButtons.forEach(b => {
            b.classList.remove('active');
            b.setAttribute('aria-checked', 'false');
        });
        btn.classList.add('active');
        btn.setAttribute('aria-checked', 'true');
        selectedLength = parseInt(btn.dataset.length);
    });
});

// Grade Button Event Listeners
gradeButtons.forEach(btn => {
    btn.addEventListener('click', () => {
        gradeButtons.forEach(b => {
            b.classList.remove('active');
            b.setAttribute('aria-checked', 'false');
        });
        btn.classList.add('active');
        btn.setAttribute('aria-checked', 'true');
        selectedGrade = btn.dataset.grade;
    });
});

// Stimmungs-Chips: schreiben ihren Wert ins Textfeld, das weiterhin frei
// editierbar bleibt (z.B. für Kombinationen wie "fröhlich und spannend")
moodChips.forEach(chip => {
    chip.addEventListener('click', () => {
        const alreadyActive = chip.classList.contains('active');
        moodChips.forEach(c => c.classList.remove('active'));
        if (alreadyActive) {
            stimmungInput.value = '';
        } else {
            chip.classList.add('active');
            stimmungInput.value = chip.dataset.mood;
        }
    });
});

// Manuelle Eingabe hebt die Chip-Auswahl auf, wenn sie nicht mehr passt
stimmungInput.addEventListener('input', () => {
    moodChips.forEach(c => {
        c.classList.toggle('active', c.dataset.mood === stimmungInput.value);
    });
});

const randomBtn = document.getElementById('random-btn');
const generateBtn = document.getElementById('generate-btn');
const backBtn = document.getElementById('back-btn');
const copyBtn = document.getElementById('copy-btn');
const printBtn = document.getElementById('print-btn');
const shareBtn = document.getElementById('share-btn');
const newStoryBtn = document.getElementById('new-story-btn');

// Pflichtfelder für die Inline-Validierung (Stil/Genre ist optional)
const requiredFields = [
    { input: themaInput, errorId: 'thema-error', message: 'Bitte gib ein Thema ein.' },
    { input: personenInput, errorId: 'personen-error', message: 'Bitte gib Personen oder Tiere ein.' },
    { input: ortInput, errorId: 'ort-error', message: 'Bitte gib einen Ort ein.' },
    { input: stimmungInput, errorId: 'stimmung-error', message: 'Bitte gib eine Stimmung ein.' }
];

function clearFieldError(field) {
    field.input.classList.remove('invalid');
    field.input.removeAttribute('aria-invalid');
    const errorEl = document.getElementById(field.errorId);
    if (errorEl) {
        errorEl.textContent = '';
        errorEl.classList.remove('visible');
    }
}

function setFieldError(field) {
    field.input.classList.add('invalid');
    field.input.setAttribute('aria-invalid', 'true');
    const errorEl = document.getElementById(field.errorId);
    if (errorEl) {
        errorEl.textContent = field.message;
        errorEl.classList.add('visible');
    }
}

// Prüft alle Pflichtfelder, markiert leere inline und gibt das erste
// ungültige Eingabefeld zurück (oder null, wenn alles ausgefüllt ist).
function validateRequiredFields() {
    let firstInvalid = null;
    requiredFields.forEach(field => {
        clearFieldError(field);
        if (!field.input.value.trim()) {
            setFieldError(field);
            if (!firstInvalid) {
                firstInvalid = field.input;
            }
        }
    });
    return firstInvalid;
}

// Fehler verschwindet, sobald das Feld ausgefüllt wird
requiredFields.forEach(field => {
    field.input.addEventListener('input', () => {
        if (field.input.value.trim()) {
            clearFieldError(field);
        }
    });
});

const storyContent = document.getElementById('story-content');
const badgeGrade = document.getElementById('badge-grade');
const badgeLength = document.getElementById('badge-length');
const storyDetails = document.getElementById('story-details');
const storyDetailsToggle = document.getElementById('story-details-toggle');
const stilRow = document.getElementById('stil-row');
const infoThema = document.getElementById('info-thema');
const infoPersonen = document.getElementById('info-personen');
const infoOrt = document.getElementById('info-ort');
const infoStimmung = document.getElementById('info-stimmung');
const infoStil = document.getElementById('info-stil');

// Details-Panel auf-/zuklappen (standardmäßig zugeklappt)
storyDetailsToggle.addEventListener('click', () => {
    storyDetails.classList.toggle('open');
});

// Event Listeners
randomBtn.addEventListener('click', getRandomSuggestions);
generateBtn.addEventListener('click', generateStory);
backBtn.addEventListener('click', showInputForm);
newStoryBtn.addEventListener('click', showInputForm);
copyBtn.addEventListener('click', copyStoryText);
printBtn.addEventListener('click', () => window.print());
shareBtn.addEventListener('click', downloadStory);

// Zufällige Vorschläge laden
async function getRandomSuggestions() {
    try {
        randomBtn.disabled = true;
        const response = await fetch(`${API_URL}/api/random`);
        const data = await response.json();
        
        themaInput.value = data.thema;
        personenInput.value = data.personen_tiere;
        ortInput.value = data.ort;
        stimmungInput.value = data.stimmung;
        stilInput.value = data.stil;

        moodChips.forEach(c => {
            c.classList.toggle('active', c.dataset.mood === data.stimmung);
        });
        requiredFields.forEach(clearFieldError);

        // Animation für visuelle Rückmeldung
        [themaInput, personenInput, ortInput, stimmungInput, stilInput].forEach(input => {
            input.style.background = 'oklch(0.94 0.03 40)';
            setTimeout(() => {
                input.style.background = '';
            }, 500);
        });
    } catch (error) {
        console.error('Fehler beim Laden der Vorschläge:', error);
        alert('Fehler beim Laden der Vorschläge. Bitte versuche es erneut.');
    } finally {
        randomBtn.disabled = false;
    }
}

// Stift-Reveal-Animation: Zeichen aus der Warteschlange werden portionsweise
// unabhängig vom Netzwerk-Timing angezeigt, damit der Text gleichmäßig
// "geschrieben" wirkt statt in Netzwerk-Bursts zu erscheinen.
// Solange der Stream noch läuft, wird mit einer sanften Grundrate enthüllt.
// Sobald der Stream fertig ist (streamComplete), wird die Portionsgröße so
// berechnet, dass der GESAMTE Rest garantiert innerhalb von
// REVEAL_MAX_CATCHUP_MS fertig angezeigt ist - unabhängig davon, wie viel
// noch übrig ist. Ohne das würde eine feste oder nur asymptotisch wachsende
// Rate bei langen Geschichten (mehrere tausend Zeichen) die Animation
// minutenlang hinter dem bereits fertigen Text herhinken lassen.
const REVEAL_BASE_CHARS_PER_TICK = 2;
const REVEAL_INTERVAL_MS = 25;
const REVEAL_MAX_CATCHUP_MS = 8000;

let revealQueue = '';
let revealTimer = null;
let streamComplete = false;
let currentParagraphEl = null;
let currentWordEl = null;
let currentAbortController = null;
let isFirstStoryChar = true;

// Geschichte generieren
async function generateStory() {
    const thema = themaInput.value.trim();
    const personen = personenInput.value.trim();
    const ort = ortInput.value.trim();
    const stimmung = stimmungInput.value.trim();
    const stil = stilInput.value.trim();
    const laenge = selectedLength;

    // Validierung: leere Pflichtfelder inline markieren und zum ersten
    // ungültigen Feld springen statt eines blockierenden alert()
    const firstInvalid = validateRequiredFields();
    if (firstInvalid) {
        firstInvalid.focus();
        return;
    }

    lastRequestParams = { thema, personen_tiere: personen, ort, stimmung };
    currentAbortController = new AbortController();

    try {
        // UI Update
        generateBtn.disabled = true;
        loading.style.display = 'flex';
        document.body.classList.add('no-scroll');
        window.scrollTo({ top: 0, behavior: 'smooth' });

        const response = await fetch(`${API_URL}/api/generate-story`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                thema: thema,
                personen_tiere: personen,
                ort: ort,
                stimmung: stimmung,
                stil: stil,
                laenge: laenge,
                klassenstufe: selectedGrade
            }),
            signal: currentAbortController.signal
        });

        if (!response.ok) {
            // Fehler vor Streaming-Start (z.B. Rate-Limit, Validierung) -
            // hier liefert der Server noch eine normale JSON-Fehlerantwort
            const data = await response.json().catch(() => ({}));
            throw new Error(data.detail || 'Fehler beim Generieren der Geschichte');
        }

        await consumeStoryStream(response);
    } catch (error) {
        if (error.name === 'AbortError') {
            return;
        }
        console.error('Fehler:', error);
        handleStreamError(error.message || 'Fehler beim Erstellen der Geschichte. Bitte versuche es erneut.');
    } finally {
        generateBtn.disabled = false;
        loading.style.display = 'none';
        document.body.classList.remove('no-scroll');
    }
}

// Liest die NDJSON-Stream-Antwort und dispatcht jedes Event
async function consumeStoryStream(response) {
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let sawTerminalEvent = false;

    try {
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop();

            for (const line of lines) {
                if (!line.trim()) continue;
                const event = JSON.parse(line);
                if (dispatchStreamEvent(event)) {
                    sawTerminalEvent = true;
                }
            }
        }

        if (buffer.trim()) {
            const event = JSON.parse(buffer);
            if (dispatchStreamEvent(event)) {
                sawTerminalEvent = true;
            }
        }

        if (!sawTerminalEvent) {
            throw new Error('Verbindung wurde unterbrochen, bevor die Geschichte fertig war.');
        }
    } finally {
        reader.releaseLock();
    }
}

// Verarbeitet ein einzelnes Stream-Event. Gibt true zurück, wenn es den
// Stream beendet (done oder error).
function dispatchStreamEvent(event) {
    switch (event.type) {
        case 'title':
            onStoryTitle(event.title);
            return false;
        case 'chunk':
            onStoryChunk(event.text);
            return false;
        case 'done':
            onStoryDone(event.grundwortschatz, event.parameters);
            return true;
        case 'error':
            throw new Error(event.detail || 'Fehler beim Erstellen der Geschichte.');
        default:
            console.warn('Unbekanntes Stream-Event:', event);
            return false;
    }
}

// Zuletzt generierte Geschichte, für den Download-als-Datei-Button
let currentStory = null;

// Eingaben der laufenden Anfrage, als Zutaten für die Cover-Illustration
let lastRequestParams = null;

// Baut die prozedurale Titelillustration aus den Eingabefeldern und dem
// Titel (als Seed) - rein clientseitig, siehe illustration.js
function renderStoryCover(title) {
    const cover = document.getElementById('story-cover');
    if (!cover || !window.mairchenIllustration) {
        return '';
    }
    const svg = window.mairchenIllustration.buildStoryIllustration({
        titel: title,
        thema: lastRequestParams ? lastRequestParams.thema : '',
        personenTiere: lastRequestParams ? lastRequestParams.personen_tiere : '',
        ort: lastRequestParams ? lastRequestParams.ort : '',
        stimmung: lastRequestParams ? lastRequestParams.stimmung : ''
    });
    cover.innerHTML = svg;
    return svg;
}

// Titel ist da: Buch aufschlagen und mit dem Reveal beginnen
function onStoryTitle(title) {
    const storyTitle = document.getElementById('story-title');
    storyTitle.textContent = title || 'Eine Geschichte';

    currentStory = { title: title || 'Eine Geschichte', text: '', parameters: null, grundwortschatz: [] };
    currentStory.coverSvg = renderStoryCover(currentStory.title);

    storyContent.innerHTML = '';
    revealQueue = '';
    currentParagraphEl = null;
    currentWordEl = null;
    isFirstStoryChar = true;
    delete storyDisplay.dataset.streamComplete;

    badgeGrade.textContent = selectedGrade === '12' ? '1./2. Klasse' : '3./4. Klasse';
    badgeLength.textContent = `${selectedLength} Min`;
    storyDetails.classList.remove('open');

    inputForm.style.display = 'none';
    loading.style.display = 'none';
    document.body.classList.remove('no-scroll');
    storyDisplay.style.display = 'block';

    window.scrollTo({ top: 0, behavior: 'smooth' });

    setTimeout(() => {
        storyDisplay.classList.remove('book-closed');
        storyDisplay.classList.add('book-opening');
    }, 100);

    startRevealLoop();
}

// Neuer Text-Chunk kommt in die Warteschlange, nicht direkt ins DOM
function onStoryChunk(text) {
    revealQueue += text;
    if (currentStory) {
        currentStory.text += text;
    }
}

// Stream fertig: Info-Panel befüllen, Reveal-Loop läuft weiter bis die
// Warteschlange leer ist
function onStoryDone(grundwortschatz, parameters) {
    if (currentStory) {
        currentStory.parameters = parameters;
        currentStory.grundwortschatz = grundwortschatz || [];
    }

    infoThema.textContent = parameters.thema;
    infoPersonen.textContent = parameters.personen_tiere;
    infoOrt.textContent = parameters.ort;
    infoStimmung.textContent = parameters.stimmung;

    // Stil/Genre: nur anzeigen, wenn vorhanden
    if (parameters.stil && parameters.stil.trim() !== '') {
        infoStil.textContent = parameters.stil;
        stilRow.style.display = '';
    } else {
        stilRow.style.display = 'none';
    }

    // Zeige Grundwortschatz-Wörter an
    const infoGrundwortschatz = document.getElementById('info-grundwortschatz');
    if (grundwortschatz && grundwortschatz.length > 0) {
        infoGrundwortschatz.textContent = grundwortschatz.join(', ');
    } else {
        infoGrundwortschatz.textContent = 'Keine gefunden';
    }

    streamComplete = true;
}

function handleStreamError(message) {
    stopRevealLoop();
    alert(message);
}

function startRevealLoop() {
    stopRevealLoop();
    streamComplete = false;
    revealTimer = setInterval(tickReveal, REVEAL_INTERVAL_MS);
}

function stopRevealLoop() {
    if (revealTimer) {
        clearInterval(revealTimer);
        revealTimer = null;
    }
}

function tickReveal() {
    if (revealQueue.length === 0) {
        if (streamComplete) {
            stopRevealLoop();
            finalizeStoryContent();
            storyDisplay.dataset.streamComplete = 'true';
        }
        return;
    }

    let charsPerTick = REVEAL_BASE_CHARS_PER_TICK;
    if (streamComplete) {
        // Kein weiterer Nachschub mehr zu erwarten - den bekannten Rest in
        // einer festen Zeitspanne abbauen, egal wie groß er ist.
        const remainingTicks = Math.max(1, Math.ceil(REVEAL_MAX_CATCHUP_MS / REVEAL_INTERVAL_MS));
        charsPerTick = Math.max(charsPerTick, Math.ceil(revealQueue.length / remainingTicks));
    }

    const take = revealQueue.slice(0, charsPerTick);
    revealQueue = revealQueue.slice(charsPerTick);
    appendRevealedText(take);
}

// Hängt ein paar Zeichen an den aktuellen Absatz an; \n schließt den
// aktuellen Absatz und öffnet beim nächsten Zeichen einen neuen. Jedes
// Zeichen bekommt einen eigenen <span> mit "Tinten-Tupfer"-Animation
// (siehe .ink-char in styles.css). Da jedes Zeichen ein eigenes
// display:inline-block-Element ist, darf der Browser sonst auch mitten in
// einem Wort umbrechen; deshalb werden die Zeichen eines Worts zusätzlich
// in einen .ink-word-Wrapper gruppiert, der als Ganzes umgebrochen wird.
function appendRevealedText(fragment) {
    for (const ch of fragment) {
        if (ch === '\n') {
            currentParagraphEl = null;
            currentWordEl = null;
            continue;
        }
        if (!currentParagraphEl) {
            currentParagraphEl = document.createElement('p');
            storyContent.appendChild(currentParagraphEl);
        }
        const isInitial = isFirstStoryChar && /\S/.test(ch);

        if (/\s/.test(ch)) {
            currentWordEl = null;
        } else if (!currentWordEl && !isInitial) {
            currentWordEl = document.createElement('span');
            currentWordEl.className = 'ink-word';
            currentParagraphEl.appendChild(currentWordEl);
        }
        const charEl = document.createElement('span');
        charEl.className = 'ink-char';
        if (isInitial) {
            // Die Initiale bekommt keinen .ink-word-Wrapper: float:left auf
            // einem inline-block-Elternteil würde sie aus dem Textfluss
            // reißen und isoliert in einer eigenen Zeile landen.
            charEl.classList.add('story-initial');
            isFirstStoryChar = false;
        }
        charEl.textContent = ch;
        (isInitial ? currentParagraphEl : (currentWordEl || currentParagraphEl)).appendChild(charEl);
    }
}

// Escaped Text für die Einbettung in HTML
function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}

// Erkennt zusammenhängende Buchstabenfolgen (inkl. Umlaute/ß dank \p{L}),
// um sie einzeln gegen die erkannten Grundwortschatz-Wörter zu prüfen.
const GWS_WORD_TOKEN_REGEX = /\p{L}+/gu;

// Das Backend erkennt ein Grundwortschatz-Wort auch als Präfix eines
// längeren Worts (Regex `\bwort\w*\b`, siehe backend/pkg/analysis), z.B.
// zählt "ab" schon als Treffer, wenn nur "abends" im Text vorkommt. Damit
// wirklich jedes vom Backend gemeldete Wort auch im Fließtext sichtbar
// hervorgehoben wird, hebt die Prüfung hier ebenso Wörter hervor, die mit
// einem der gemeldeten Grundwortschatz-Wörter beginnen.
function buildGwsMatcher(words) {
    const lowerWords = (words || []).map(w => w.toLowerCase());
    if (lowerWords.length === 0) {
        return null;
    }
    return (word) => {
        const lower = word.toLowerCase();
        return lowerWords.some(w => lower.startsWith(w));
    };
}

// Ersetzt Grundwortschatz-Treffer in rohem (noch nicht escapetem) Text durch
// <mark>-Elemente. Escaping passiert stückweise für Treffer und Lücken
// dazwischen, damit &/</> im Story-Text nie ungeescaped ins Markup gelangen.
function highlightGrundwortschatzWords(text, isGwsMatch) {
    if (!isGwsMatch) {
        return escapeHtml(text);
    }
    let html = '';
    let lastIndex = 0;
    for (const match of text.matchAll(GWS_WORD_TOKEN_REGEX)) {
        const word = match[0];
        html += escapeHtml(text.slice(lastIndex, match.index));
        html += isGwsMatch(word)
            ? `<mark class="gws-highlight">${escapeHtml(word)}</mark>`
            : escapeHtml(word);
        lastIndex = match.index + word.length;
    }
    html += escapeHtml(text.slice(lastIndex));
    return html;
}

// Baut das finale (nicht mehr animierte) Story-Markup: Absätze mit
// hervorgehobenen Grundwortschatz-Wörtern, plus der große Initialbuchstabe
// auf dem ersten Absatz - wie zuvor durch die Zeichen-Reveal-Animation.
function buildStoryContentHtml(text, grundwortschatzWords) {
    const isGwsMatch = buildGwsMatcher(grundwortschatzWords);
    const paragraphs = text.split('\n').map(p => p.trim()).filter(Boolean);
    return paragraphs.map((paragraph, index) => {
        if (index === 0) {
            const firstChar = paragraph.slice(0, 1);
            const rest = paragraph.slice(1);
            return `<p><span class="story-initial">${escapeHtml(firstChar)}</span>${highlightGrundwortschatzWords(rest, isGwsMatch)}</p>`;
        }
        return `<p>${highlightGrundwortschatzWords(paragraph, isGwsMatch)}</p>`;
    }).join('');
}

// Ersetzt die Zeichen-für-Zeichen animierten Spans durch das finale,
// hervorgehobene Markup, sobald der Reveal fertig ist - die Tinten-Tupfer-
// Animation ist zu diesem Zeitpunkt für jedes Zeichen bereits abgespielt.
function finalizeStoryContent() {
    if (!currentStory) {
        return;
    }
    storyContent.innerHTML = buildStoryContentHtml(currentStory.text, currentStory.grundwortschatz);
}

// Icon-Markup des Kopieren-Buttons, um nach dem Kopieren kurz auf ein
// Häkchen umzuschalten und danach wieder zurückzuwechseln
const copyBtnIconHtml = copyBtn.innerHTML;
const copyBtnCheckIconHtml = '<svg width="14" height="14" viewBox="0 0 14 14"><path d="M2.5 7.5l3 3 6-6.5" stroke="currentColor" stroke-width="1.6" fill="none" stroke-linecap="round" stroke-linejoin="round"/></svg>';
let copyFeedbackTimeout = null;

// Kopiert Titel und Text der Geschichte als Plain Text in die
// Zwischenablage, mit Fallback für unsichere Kontexte ohne Clipboard-API
async function copyStoryText() {
    if (!currentStory || !currentStory.parameters) {
        return;
    }
    const text = `${currentStory.title}\n\n${currentStory.text.trim()}`;
    try {
        if (navigator.clipboard && window.isSecureContext) {
            await navigator.clipboard.writeText(text);
        } else {
            copyTextFallback(text);
        }
        showCopyFeedback();
    } catch (err) {
        console.error('Kopieren fehlgeschlagen:', err);
    }
}

function copyTextFallback(text) {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
}

function showCopyFeedback() {
    clearTimeout(copyFeedbackTimeout);
    copyBtn.innerHTML = copyBtnCheckIconHtml;
    copyBtn.classList.add('copied');
    copyBtn.setAttribute('aria-label', 'In die Zwischenablage kopiert');
    copyFeedbackTimeout = setTimeout(() => {
        copyBtn.innerHTML = copyBtnIconHtml;
        copyBtn.classList.remove('copied');
        copyBtn.setAttribute('aria-label', 'Geschichte in die Zwischenablage kopieren');
    }, 1600);
}

// Baut aus der aktuellen Geschichte eine eigenständige HTML-Datei im
// mAIrchen-Design und lädt sie herunter - kein Backend-Endpoint und keine
// öffentlichen Links nötig, die App bleibt intern.
function downloadStory() {
    if (!currentStory || !currentStory.parameters) {
        return;
    }

    const paragraphs = currentStory.text
        .split('\n')
        .map(p => p.trim())
        .filter(Boolean)
        .map(p => `<p>${escapeHtml(p)}</p>`)
        .join('\n');

    const p = currentStory.parameters;
    const gradeLabel = p.klassenstufe === '12' ? '1./2. Klasse' : '3./4. Klasse';
    const stilRowHtml = p.stil && p.stil.trim() !== ''
        ? `<div class="story-details-item"><div class="label">Stil/Genre</div><div class="value">${escapeHtml(p.stil)}</div></div>`
        : '';
    const gwsText = currentStory.grundwortschatz.length > 0
        ? currentStory.grundwortschatz.join(', ')
        : 'Keine gefunden';

    // Cover-SVG ist selbst erzeugtes Markup (keine Nutzereingaben), kann
    // also direkt eingebettet werden
    const coverHtml = currentStory.coverSvg
        ? `<div class="cover">${currentStory.coverSvg}</div>`
        : '';

    const html = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<title>${escapeHtml(currentStory.title)} - mAIrchen</title>
<style>
@import url('https://fonts.googleapis.com/css2?family=Quicksand:wght@600;700&family=Karla:wght@400;600&display=swap');
body{margin:0;background:oklch(0.97 0.012 68);color:oklch(0.26 0.02 50);font-family:'Karla',sans-serif;padding:40px 20px;}
.wrap{max-width:640px;margin:0 auto;}
.badges{display:flex;gap:8px;margin-bottom:16px;}
.badge{font-family:'Karla',sans-serif;font-weight:600;font-size:0.75rem;padding:4px 12px;border-radius:999px;}
.badge.grade{color:oklch(0.5 0.14 35);background:oklch(0.94 0.03 40);}
.badge.length{color:oklch(0.42 0.09 322);background:oklch(0.93 0.03 322);}
h1{font-family:'Quicksand',sans-serif;font-weight:700;font-size:1.8rem;margin:0 0 20px;padding-bottom:16px;border-bottom:1px solid oklch(0.89 0.016 60);}
.story-text{font-size:1.1rem;line-height:1.8;}
.story-text p{margin:0 0 20px;}
.details{margin-top:32px;padding-top:16px;border-top:1px solid oklch(0.89 0.016 60);display:grid;grid-template-columns:1fr 1fr;gap:14px 20px;}
.story-details-item .label{font-family:'Quicksand',sans-serif;font-weight:600;font-size:0.72rem;color:oklch(0.62 0.02 50);text-transform:uppercase;letter-spacing:.04em;margin-bottom:4px;}
.story-details-item .value{font-size:0.9rem;}
.cover{height:170px;border-radius:18px;overflow:hidden;margin-bottom:20px;}
.cover svg{width:100%;height:100%;display:block;}
</style>
</head>
<body>
<div class="wrap">
<div class="badges">
<span class="badge grade">${escapeHtml(gradeLabel)}</span>
<span class="badge length">${escapeHtml(String(p.laenge))} Min</span>
</div>
${coverHtml}
<h1>${escapeHtml(currentStory.title)}</h1>
<div class="story-text">
${paragraphs}
</div>
<div class="details">
<div class="story-details-item"><div class="label">Thema</div><div class="value">${escapeHtml(p.thema)}</div></div>
<div class="story-details-item"><div class="label">Personen/Tiere</div><div class="value">${escapeHtml(p.personen_tiere)}</div></div>
<div class="story-details-item"><div class="label">Ort</div><div class="value">${escapeHtml(p.ort)}</div></div>
<div class="story-details-item"><div class="label">Stimmung</div><div class="value">${escapeHtml(p.stimmung)}</div></div>
${stilRowHtml}
<div class="story-details-item"><div class="label">Grundwortschatz-Wörter</div><div class="value">${escapeHtml(gwsText)}</div></div>
</div>
</div>
</body>
</html>
`;

    const blob = new Blob([html], { type: 'text/html' });
    const url = URL.createObjectURL(blob);
    const filename = currentStory.title
        .toLowerCase()
        .replace(/[^a-z0-9äöüß]+/g, '-')
        .replace(/^-+|-+$/g, '') || 'geschichte';

    const a = document.createElement('a');
    a.href = url;
    a.download = `${filename}.html`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
}

// Zurück zum Formular
function showInputForm() {
    if (currentAbortController) {
        currentAbortController.abort();
    }
    stopRevealLoop();
    storyDisplay.style.display = 'none';
    storyDisplay.classList.remove('book-opening');
    storyDisplay.classList.add('book-closed');
    inputForm.style.display = 'block';
    window.scrollTo({ top: 0, behavior: 'smooth' });
}

// Initiale Ladung - überprüfe API-Verbindung
async function checkAPIConnection() {
    try {
        const response = await fetch(`${API_URL}/health`);
        if (!response.ok) {
            console.warn('API ist nicht erreichbar');
        }
    } catch (error) {
        console.warn('API-Verbindung konnte nicht hergestellt werden:', error);
    }
}

// Beim Laden der Seite
checkAPIConnection();
