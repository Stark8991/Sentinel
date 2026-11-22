package main

import (
	"sync"
	"time"
)

type Decision string

const (
	Allow  Decision = "ALLOW"
	Review Decision = "REVIEW"
	Block  Decision = "BLOCK"
)

const (
	blockScore  = 70
	reviewScore = 40
)

// Result is the engine's verdict on one transaction.
type Result struct {
	Txn      Transaction
	Score    int
	Decision Decision
	Hits     []Hit
	Latency  time.Duration
}

// Stats are running totals, including how well planted fraud is being caught.
type Stats struct {
	Processed, Allowed, Reviewed, Blocked int
	PlantedTotal, PlantedCaught           int
	FalseAlarms                           int // flagged but not planted
	TotalLatency                          time.Duration
}

type Engine struct {
	rules   []Rule
	history *History
	mu      sync.Mutex
	stats   Stats
}

func NewEngine(rules []Rule) *Engine {
	return &Engine{rules: rules, history: NewHistory()}
}

// Evaluate scores one transaction, records it in history and updates stats.
func (e *Engine) Evaluate(t Transaction) Result {
	start := time.Now()
	past := e.history.Get(t.CardID)

	res := Result{Txn: t, Decision: Allow}
	for _, rule := range e.rules {
		if hit, ok := rule(t, past); ok {
			res.Hits = append(res.Hits, hit)
			res.Score += hit.Points
		}
	}
	switch {
	case res.Score >= blockScore:
		res.Decision = Block
	case res.Score >= reviewScore:
		res.Decision = Review
	}
	e.history.Add(t)
	res.Latency = time.Since(start)

	e.record(res)
	return res
}

func (e *Engine) record(r Result) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := &e.stats
	s.Processed++
	s.TotalLatency += r.Latency
	switch r.Decision {
	case Allow:
		s.Allowed++
	case Review:
		s.Reviewed++
	case Block:
		s.Blocked++
	}
	flagged := r.Decision != Allow
	if r.Txn.Planted != "" {
		s.PlantedTotal++
		if flagged {
			s.PlantedCaught++
		}
	} else if flagged {
		s.FalseAlarms++
	}
}

// Snapshot returns a copy of the current stats.
func (e *Engine) Snapshot() Stats {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stats
}
