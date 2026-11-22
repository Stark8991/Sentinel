package main

import (
	"testing"
	"time"
)

func TestEngineDecisions(t *testing.T) {
	cases := []struct {
		name      string
		txn       Transaction
		wantScore int
		wantDec   Decision
	}{
		{"normal payment is allowed", txn(800, "food", "Chennai", base), 0, Allow},
		{"crypto payment goes to review", txn(800, "crypto", "Chennai", base), 40, Review},
		{"big new-card payment at 3 am is blocked",
			txn(60000, "shopping", "Chennai", time.Date(2026, 11, 22, 3, 0, 0, 0, time.Local)), 90, Block},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := NewEngine(DefaultRules) // fresh engine = empty history
			r := e.Evaluate(c.txn)
			if r.Score != c.wantScore || r.Decision != c.wantDec {
				t.Errorf("got score %d %s, want %d %s (hits: %+v)", r.Score, r.Decision, c.wantScore, c.wantDec, r.Hits)
			}
		})
	}
}

func TestEngineStatsCountPlantedFraud(t *testing.T) {
	e := NewEngine(DefaultRules)
	normal := txn(800, "food", "Chennai", base)
	fraud := txn(800, "crypto", "Chennai", base)
	fraud.Planted = "risky-merchant"
	e.Evaluate(normal)
	e.Evaluate(fraud)

	s := e.Snapshot()
	if s.Processed != 2 || s.PlantedTotal != 1 || s.PlantedCaught != 1 || s.FalseAlarms != 0 {
		t.Errorf("unexpected stats: %+v", s)
	}
}
