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
        
        // Animation für visuelle Rückmeldung
        [themaInput, personenInput, ortInput, stimmungInput].forEach(input => {
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

// Geschichte generieren
async function generateStory() {
    const thema = themaInput.value.trim();
    const personen = personenInput.value.trim();
    const ort = ortInput.value.trim();
    const stimmung = stimmungInput.value.trim();
    const laenge = selectedLength;
    
    // Validierung
    if (!thema || !personen || !ort || !stimmung) {
        alert('Bitte fülle alle Felder aus!');
        return;
    }
    
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
                laenge: laenge,
                klassenstufe: selectedGrade
            })
        });
        
        if (!response.ok) {
            throw new Error('Fehler beim Generieren der Geschichte');
        }
        
        const data = await response.json();
        
        if (data.success) {
            displayStory(data.story, data.title, data.parameters);
        } else {
            throw new Error('Keine Geschichte erhalten');
        }
    } catch (error) {
        console.error('Fehler:', error);
        alert('Fehler beim Erstellen der Geschichte. Bitte versuche es erneut.');
    } finally {
        generateBtn.disabled = false;
        loading.style.display = 'none';
    }
}

// Geschichte anzeigen
function displayStory(story, title, parameters) {
    // Formatiere die Geschichte mit Absätzen
    const formattedStory = formatStoryText(story);
    
    // Setze Titel
    const storyTitle = document.getElementById('story-title');
    storyTitle.textContent = title || 'Eine Geschichte';
    
    storyContent.innerHTML = formattedStory;
    infoThema.textContent = parameters.thema;
    infoPersonen.textContent = parameters.personen_tiere;
    infoOrt.textContent = parameters.ort;
    infoStimmung.textContent = parameters.stimmung;
    
    // Ansicht wechseln
    inputForm.style.display = 'none';
    storyDisplay.style.display = 'block';
    
    // Scrolle nach oben
    window.scrollTo({ top: 0, behavior: 'smooth' });
    
    // Trigger Buch-Öffnungs-Animation
    setTimeout(() => {
        storyDisplay.classList.remove('book-closed');
        storyDisplay.classList.add('book-opening');
    }, 100);
}

// Text formatieren
function formatStoryText(text) {
    // Teile den Text in Absätze
    const paragraphs = text.split('\n').filter(p => p.trim().length > 0);
    
    // Erstelle HTML mit Absätzen
    return paragraphs.map(p => `<p>${p.trim()}</p>`).join('');
}

// Zurück zum Formular
function showInputForm() {
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
