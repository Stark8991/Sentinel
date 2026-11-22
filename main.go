package main

import (
	"fmt"
	"strings"
	"time"
)

func main() {
	fmt.Println("Sentinel fraud engine starting... (Ctrl+C to stop)")

	txns := make(chan Transaction, 100)
	gen := NewGenerator(2000) // enough cards that normal customers don't look like a "burst"
	go gen.Run(txns, 20)      // ~20 transactions per second

	engine := NewEngine(DefaultRules)
	hub := NewHub()
	go hub.PublishStats(engine, time.Second)
	go printStats(engine, 5*time.Second)
	go StartServer(":8080", hub)

	for t := range txns {
		r := engine.Evaluate(t)
		if r.Decision != Allow {
			hub.PublishAlert(r)
			fmt.Println(formatResult(r))
		}
	}
}

func formatResult(r Result) string {
	icon := "🟡"
	if r.Decision == Block {
		icon = "🔴"
	}
	reasons := make([]string, len(r.Hits))
	for i, h := range r.Hits {
		reasons[i] = fmt.Sprintf("%s +%d (%s)", h.Rule, h.Points, h.Reason)
	}
	return fmt.Sprintf("%s %-6s %s  %s  ₹%9.2f  %-10s %-10s score %3d  [%s]",
		icon, r.Decision, r.Txn.ID, r.Txn.CardID, r.Txn.Amount, r.Txn.Merchant, r.Txn.City, r.Score, strings.Join(reasons, ", "))
}

func printStats(e *Engine, every time.Duration) {
	for range time.Tick(every) {
		s := e.Snapshot()
		avg := 0.0
		if s.Processed > 0 {
			avg = float64(s.TotalLatency.Nanoseconds()) / float64(s.Processed) / 1e6
		}
		caught := 0.0
		if s.PlantedTotal > 0 {
			caught = 100 * float64(s.PlantedCaught) / float64(s.PlantedTotal)
		}
		fmt.Printf("[stats] %d processed | %d allow | %d review | %d block | planted fraud caught %d/%d (%.0f%%) | false alarms %d | avg %.3f ms/txn\n",
			s.Processed, s.Allowed, s.Reviewed, s.Blocked, s.PlantedCaught, s.PlantedTotal, caught, s.FalseAlarms, avg)
	}
}
