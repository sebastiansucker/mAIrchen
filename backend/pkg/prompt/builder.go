package prompt

import (
	"fmt"

	"github.com/sebastiansucker/mAIrchen/backend/pkg/data"
)

// StoryRequest represents the parameters for story generation
type StoryRequest struct {
	Thema          string `json:"thema"`
	PersonenTiere  string `json:"personen_tiere"`
	Ort            string `json:"ort"`
	Stimmung       string `json:"stimmung"`
	Laenge         int    `json:"laenge"`
	Klassenstufe   string `json:"klassenstufe"`
	Stil           string `json:"stil,omitempty"`
	Model          string `json:"model,omitempty"`
}

// BuildPrompt creates the system and user prompts for story generation
func BuildPrompt(req StoryRequest) (string, string) {
	var minWords, maxWords int
	var zielgruppe, schwierigkeit, grundwortschatz string
	
	if req.Klassenstufe == "12" {
		minWords = req.Laenge * 60
		maxWords = req.Laenge * 70
		zielgruppe = "Kinder der Klassenstufen 1 & 2"
		schwierigkeit = "sehr einfach mit kurzen Sätzen und einfachen Wörtern"
		
		// Extract Klasse 1-2 section
		parts := splitGWSContent()
		grundwortschatz = parts[0]
	} else {
		minWords = req.Laenge * 80
		maxWords = req.Laenge * 100
		zielgruppe = "Kinder der Klassenstufen 3 & 4"
		schwierigkeit = "kindgerecht mit etwas längeren Sätzen und anspruchsvolleren Wörtern"
		grundwortschatz = data.GrundwortschatzContent
	}
	
	stilInstruction := ""
	if req.Stil != "" {
		stilInstruction = fmt.Sprintf("- Stil/Genre: %s\n", req.Stil)
	}
	
	systemPrompt := fmt.Sprintf("Du bist ein kreativer Geschichtenerzähler für %s.", zielgruppe)
	
	userPrompt := fmt.Sprintf(`Schreibe eine Geschichte mit folgenden Eigenschaften:
- Lesezeit: etwa %d Minuten (ca. %d-%d Wörter)
- Thema: %s
- Personen/Tiere: %s
- Ort: %s
- Stimmung: %s
%s- Schwierigkeitsgrad: %s

WICHTIG: Verwende beim Schreiben häufig Wörter aus dem Grundwortschatz als Leseübung.
Die Geschichte sollte kindgerecht, spannend und lehrreich sein.
Schreibe die Geschichte in normalem Text ohne Markdown-Formatierung (keine **fett** markierten Wörter).

Hier ist der Grundwortschatz zur Orientierung:
%s

Format:
Gib die Antwort im folgenden Format zurück:
TITEL: [Ein kurzer, ansprechender Titel für die Geschichte]

[Die Geschichte in Absätzen]

Beginne direkt mit "TITEL:" gefolgt vom Titel.

WICHTIG: Schreibe wirklich die vollständige Geschichte mit ca. %d Wörtern. Mache die Geschichte nicht kürzer!`,
		req.Laenge, minWords, maxWords,
		req.Thema, req.PersonenTiere, req.Ort, req.Stimmung,
		stilInstruction, schwierigkeit,
		grundwortschatz,
		maxWords)
	
	return systemPrompt, userPrompt
}

func splitGWSContent() []string {
	parts := []string{data.GrundwortschatzContent}
	separator := "### **Grundwortschatz für Jahrgangsstufen 3 und 4**"
	idx := 0
	for i := range data.GrundwortschatzContent {
		if i+len(separator) <= len(data.GrundwortschatzContent) && data.GrundwortschatzContent[i:i+len(separator)] == separator {
			idx = i
			break
		}
	}
	if idx > 0 {
		parts = []string{data.GrundwortschatzContent[:idx], data.GrundwortschatzContent[idx:]}
	}
	return parts
}

// GetGWSContent returns the embedded Grundwortschatz content
func GetGWSContent() string {
	return data.GrundwortschatzContent
}
