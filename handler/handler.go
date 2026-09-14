package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PercyJax/handy-router/actions"
	"github.com/PercyJax/handy-router/config"
	"github.com/PercyJax/handy-router/llm"
	"github.com/PercyJax/handy-router/router"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

type ChatCompletionResponse struct {
	Choices []ChatChoice `json:"choices"`
}

var transcriptRe = regexp.MustCompile(`(?s)<transcript>\s*(.*?)\s*</transcript>`)

// extractTranscription pulls the raw text from between <transcript> tags
// if present (Handy legacy mode). Otherwise returns the input as-is.
func extractTranscription(s string) string {
	if m := transcriptRe.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return s
}

type HistoryEntry struct {
	Timestamp time.Time
	Route     string
	Query     string
}

type Handler struct {
	Config  *config.Config
	History []HistoryEntry
	Mu      sync.RWMutex
}

func New(cfg *config.Config) *Handler {
	return &Handler{
		Config:  cfg,
		History: make([]HistoryEntry, 0, 20),
	}
}

func (h *Handler) addHistory(route, query string) {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	h.History = append(h.History, HistoryEntry{
		Timestamp: time.Now(),
		Route:     route,
		Query:     query,
	})

	// Keep last 20 entries
	if len(h.History) > 20 {
		h.History = h.History[len(h.History)-20:]
	}
}

func (h *Handler) GetHistory() []HistoryEntry {
	h.Mu.RLock()
	defer h.Mu.RUnlock()

	result := make([]HistoryEntry, len(h.History))
	copy(result, h.History)
	return result
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/" {
		h.serveStatusPage(w, r)
		return
	}

	if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
		http.NotFound(w, r)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract transcription from user message (last message)
	transcription := ""
	if len(req.Messages) > 0 {
		transcription = req.Messages[len(req.Messages)-1].Content
	}

	// Handy legacy mode sends the full prompt template with <transcript> tags.
	// Extract just the transcription from between the tags.
	transcription = extractTranscription(transcription)

	log.Printf("Received transcription: %q", transcription)

	// Parse route
	route := router.ParseRoute(transcription, h.Config.Routes.Prefixes)
	log.Printf("Route: %s, Query: %q", route.Name, route.Query)

	// Execute action
	var responseContent string

	switch route.Name {
	case "gemini":
		go func() {
			if err := actions.OpenGemini(route.Query, h.Config.Gemini.URLTemplate, h.Config.Gemini.Browser, h.Config.Gemini.BrowserArgs); err != nil {
				log.Printf("Failed to open Gemini: %v", err)
			}
		}()
		responseContent = ""
		h.addHistory("gemini", route.Query)

	case "code":
		go func() {
			if err := actions.OpenOpenCode(
				route.Query,
				h.Config.OpenCode.Terminal,
				h.Config.OpenCode.TerminalArgs,
				h.Config.OpenCode.Binary,
				h.Config.OpenCode.ExtraArgs,
				h.Config.OpenCode.ServerURL,
				h.Config.OpenCode.Dir,
			); err != nil {
				log.Printf("Failed to open OpenCode: %v", err)
			}
		}()
		responseContent = ""
		h.addHistory("code", route.Query)

	case "enhance":
		query := route.Query
		if h.Config.Enhance.Enabled {
			enhanced, err := llm.Enhance(query, h.Config.Enhance.Endpoint, h.Config.Enhance.Model, h.Config.Enhance.APIKey)
			if err != nil {
				log.Printf("Enhancement failed, using raw text: %v", err)
			} else {
				query = enhanced
				log.Printf("Enhanced: %q -> %q", route.Query, query)
			}
		}

		// Inject the enhanced text directly into the active window so it works
		// in terminals and text boxes alike. Handy's paste method must be set
		// to "None" so it does not paste a second time; the text is still
		// returned below so Handy can place it on the clipboard.
		if h.Config.Enhance.Paste {
			if err := actions.TypeText(query, h.Config.Enhance.PasteBinary); err != nil {
				log.Printf("Failed to type text into active window: %v", err)
			} else {
				log.Printf("Typed enhanced text into active window")
			}
		}

		responseContent = query
		h.addHistory("enhance", route.Query)

	default:
		log.Printf("No wake word matched, no-op")
		responseContent = ""
		h.addHistory("default", route.Query)
	}

	resp := ChatCompletionResponse{
		Choices: []ChatChoice{
			{
				Message: ChatMessage{
					Role:    "assistant",
					Content: responseContent,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) serveStatusPage(w http.ResponseWriter, r *http.Request) {
	history := h.GetHistory()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<title>handy-router</title>
<style>
body { font-family: monospace; background: #1a1a2e; color: #e0e0e0; padding: 2rem; }
h1 { color: #00d4ff; }
h2 { color: #7b68ee; }
table { border-collapse: collapse; width: 100%%; }
th, td { text-align: left; padding: 0.5rem 1rem; border-bottom: 1px solid #333; }
th { color: #00d4ff; }
.route { color: #7b68ee; font-weight: bold; }
.query { color: #e0e0e0; }
.status { color: #00ff88; }
</style>
</head>
<body>
<h1>handy-router</h1>
<p class="status">Running on %s:%d</p>

<h2>Routes</h2>
<table>
<tr><th>Prefix</th><th>Destination</th></tr>
<tr><td class="route">gemini</td><td>Browser (Gemini)</td></tr>
<tr><td class="route">code</td><td>Terminal (OpenCode)</td></tr>
<tr><td class="route">(none)</td><td>Default paste</td></tr>
</table>

<h2>Recent History</h2>
<table>
<tr><th>Time</th><th>Route</th><th>Query</th></tr>
`, h.Config.Server.Host, h.Config.Server.Port)

	for i := len(history) - 1; i >= 0; i-- {
		entry := history[i]
		fmt.Fprintf(w, "<tr><td>%s</td><td class=\"route\">%s</td><td class=\"query\">%s</td></tr>\n",
			entry.Timestamp.Format("15:04:05"),
			entry.Route,
			entry.Query,
		)
	}

	fmt.Fprintf(w, `</table>
</body>
</html>`)
}
