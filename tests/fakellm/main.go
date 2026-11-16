// Fake OpenAI-compatible chat completion server for the E2E suite.
// It always streams the same canned story, so the Playwright tests exercise
// the real backend/frontend flow without a paid LLM or an API key.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Uses the same TITEL:/ENDE markers the real prompt asks for, and plenty of
// Grundwortschatz words so the word list in the UI is non-empty.
const story = `TITEL: Der kleine Hase und der Fuchs

Es war einmal ein kleiner Hase. Er wohnte in einem großen Wald und hatte viele Freunde.
Jeden Morgen lief er über die Wiese und suchte nach Blumen. Die Sonne schien warm, und die Vögel sangen im Baum.

Eines Tages traf der Hase einen Fuchs. Der Fuchs war allein und sehr traurig. "Warum bist du so traurig?", fragte der Hase.
"Ich habe keine Freunde", sagte der Fuchs leise. Da lachte der Hase und sagte: "Dann spielen wir zusammen!"

Von diesem Tag an spielten der Hase und der Fuchs jeden Nachmittag am Bach. Sie bauten ein Haus aus Ästen und Blättern.
Am Abend gingen beide müde und glücklich nach Hause. Und wenn sie nicht gestorben sind, dann spielen sie noch heute.

ENDE`

type chunk struct {
	Choices []map[string]any `json:"choices"`
	Usage   map[string]int   `json:"usage,omitempty"`
}

func main() {
	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		send := func(c chunk) {
			b, _ := json.Marshal(c)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
			w.(http.Flusher).Flush()
		}
		// One chunk per line, like a real model streaming tokens.
		for _, line := range strings.SplitAfter(story, "\n") {
			send(chunk{Choices: []map[string]any{{"index": 0, "delta": map[string]string{"content": line}}}})
		}
		send(chunk{Choices: []map[string]any{}, Usage: map[string]int{"prompt_tokens": 100, "completion_tokens": 200, "total_tokens": 300}})
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	})
	log.Println("fakellm listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
