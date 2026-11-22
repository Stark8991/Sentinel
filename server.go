package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"
)

//go:embed web
var webFiles embed.FS

// Hub fans out events to every connected browser using Server-Sent Events.
type Hub struct {
	mu      sync.Mutex
	clients map[chan string]bool
	recent  []string // last alerts, replayed to new browsers
}

func NewHub() *Hub { return &Hub{clients: make(map[chan string]bool)} }

func (h *Hub) broadcast(event string, data any, keep bool) {
	b, _ := json.Marshal(data)
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, b)
	h.mu.Lock()
	defer h.mu.Unlock()
	if keep {
		h.recent = append(h.recent, msg)
		if len(h.recent) > 50 {
			h.recent = h.recent[1:]
		}
	}
	for c := range h.clients {
		select {
		case c <- msg:
		default: // slow browser: drop this event rather than block the engine
		}
	}
}

func (h *Hub) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	c := make(chan string, 64)
	h.mu.Lock()
	h.clients[c] = true
	backlog := append([]string(nil), h.recent...)
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.clients, c); h.mu.Unlock() }()

	for _, m := range backlog {
		fmt.Fprint(w, m)
	}
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case m := <-c:
			fmt.Fprint(w, m)
			flusher.Flush()
		}
	}
}

// alertView is the JSON sent to the browser for each flagged transaction.
type alertView struct {
	ID, Card, Merchant, Category, City, Time, Decision string
	Amount                                             float64
	Score                                              int
	Reasons                                            []string
	LatencyUs                                          int64
}

func (h *Hub) PublishAlert(r Result) {
	reasons := make([]string, len(r.Hits))
	for i, hit := range r.Hits {
		reasons[i] = fmt.Sprintf("%s +%d · %s", hit.Rule, hit.Points, hit.Reason)
	}
	h.broadcast("alert", alertView{
		ID: r.Txn.ID, Card: r.Txn.CardID, Merchant: r.Txn.Merchant, Category: r.Txn.Category,
		City: r.Txn.City, Time: r.Txn.Time.Format("15:04:05"), Decision: string(r.Decision),
		Amount: r.Txn.Amount, Score: r.Score, Reasons: reasons, LatencyUs: r.Latency.Microseconds(),
	}, true)
}

// PublishStats sends the running totals to all browsers every `every`.
func (h *Hub) PublishStats(e *Engine, every time.Duration) {
	start := time.Now()
	for range time.Tick(every) {
		s := e.Snapshot()
		avgMs := 0.0
		if s.Processed > 0 {
			avgMs = float64(s.TotalLatency.Nanoseconds()) / float64(s.Processed) / 1e6
		}
		h.broadcast("stats", map[string]any{
			"processed": s.Processed, "allowed": s.Allowed, "reviewed": s.Reviewed, "blocked": s.Blocked,
			"plantedTotal": s.PlantedTotal, "plantedCaught": s.PlantedCaught, "falseAlarms": s.FalseAlarms,
			"avgMs": avgMs, "tps": float64(s.Processed) / time.Since(start).Seconds(),
		}, false)
	}
}

func StartServer(addr string, h *Hub) {
	static, _ := fs.Sub(webFiles, "web")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/events", h.handleEvents)
	log.Printf("Dashboard: http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
