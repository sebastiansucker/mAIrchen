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
let selectedLength = 10; // Standard: 10 Minuten
let selectedGrade = '34'; // Standard: 3/4 Klasse

// Length Button Event Listeners
lengthButtons.forEach(btn => {
    btn.addEventListener('click', () => {
        lengthButtons.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        selectedLength = parseInt(btn.dataset.length);
    });
});

// Grade Button Event Listeners
gradeButtons.forEach(btn => {
    btn.addEventListener('click', () => {
        gradeButtons.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        selectedGrade = btn.dataset.grade;
    });
});

const randomBtn = document.getElementById('random-btn');
const generateBtn = document.getElementById('generate-btn');
const backBtn = document.getElementById('back-btn');

const storyContent = document.getElementById('story-content');
const infoThema = document.getElementById('info-thema');
const infoPersonen = document.getElementById('info-personen');
const infoOrt = document.getElementById('info-ort');
const infoStimmung = document.getElementById('info-stimmung');
const infoStil = document.getElementById('info-stil');

// Event Listeners
randomBtn.addEventListener('click', getRandomSuggestions);
generateBtn.addEventListener('click', generateStory);
backBtn.addEventListener('click', showInputForm);

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
        
        // Animation für visuelle Rückmeldung
        [themaInput, personenInput, ortInput, stimmungInput, stilInput].forEach(input => {
            input.style.background = '#e0e7ff';
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
let currentAbortController = null;

// Geschichte generieren
async function generateStory() {
    const thema = themaInput.value.trim();
    const personen = personenInput.value.trim();
    const ort = ortInput.value.trim();
    const stimmung = stimmungInput.value.trim();
    const stil = stilInput.value.trim();
    const laenge = selectedLength;

    // Validierung
    if (!thema || !personen || !ort || !stimmung) {
        alert('Bitte fülle alle Pflichtfelder aus!');
        return;
    }

    currentAbortController = new AbortController();

    try {
        // UI Update
        generateBtn.disabled = true;
        loading.style.display = 'block';

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

// Titel ist da: Buch aufschlagen und mit dem Reveal beginnen
function onStoryTitle(title) {
    const storyTitle = document.getElementById('story-title');
    storyTitle.textContent = title || 'Eine Geschichte';

    storyContent.innerHTML = '';
    revealQueue = '';
    currentParagraphEl = null;
    delete storyDisplay.dataset.streamComplete;

    inputForm.style.display = 'none';
    loading.style.display = 'none';
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
}

// Stream fertig: Info-Panel befüllen, Reveal-Loop läuft weiter bis die
// Warteschlange leer ist
function onStoryDone(grundwortschatz, parameters) {
    infoThema.textContent = parameters.thema;
    infoPersonen.textContent = parameters.personen_tiere;
    infoOrt.textContent = parameters.ort;
    infoStimmung.textContent = parameters.stimmung;

    // Stil/Genre: nur anzeigen, wenn vorhanden
    const stilRow = infoStil.parentElement;
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
// (siehe .ink-char in styles.css).
function appendRevealedText(fragment) {
    for (const ch of fragment) {
        if (ch === '\n') {
            currentParagraphEl = null;
            continue;
        }
        if (!currentParagraphEl) {
            currentParagraphEl = document.createElement('p');
            storyContent.appendChild(currentParagraphEl);
        }
        const charEl = document.createElement('span');
        charEl.className = 'ink-char';
        charEl.textContent = ch;
        currentParagraphEl.appendChild(charEl);
    }
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
