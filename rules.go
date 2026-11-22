package main

import (
	"fmt"
	"time"
)

// Hit is what a rule returns when it fires.
type Hit struct {
	Rule   string
	Points int
	Reason string
}

// Rule checks one transaction against the card's recent history (oldest first).
// It returns (hit, true) if the rule fires.
type Rule func(t Transaction, past []Transaction) (Hit, bool)

// DefaultRules is the list the engine runs. Add a new rule by writing a function and listing it here.
var DefaultRules = []Rule{VelocityRule, ImpossibleTravelRule, AmountSpikeRule, RiskyMerchantRule, NightSpendRule}

// VelocityRule: 5 or more transactions on this card within the last 60 seconds (including this one).
func VelocityRule(t Transaction, past []Transaction) (Hit, bool) {
	count := 1
	for _, p := range past {
		if t.Time.Sub(p.Time) <= 60*time.Second && t.Time.Sub(p.Time) >= 0 {
			count++
		}
	}
	if count >= 5 {
		return Hit{"velocity", 40, fmt.Sprintf("%d txns in 60s", count)}, true
	}
	return Hit{}, false
}

// ImpossibleTravelRule: the previous transaction was in a different city less than an hour ago.
func ImpossibleTravelRule(t Transaction, past []Transaction) (Hit, bool) {
	if len(past) == 0 {
		return Hit{}, false
	}
	prev := past[len(past)-1]
	gap := t.Time.Sub(prev.Time)
	if prev.City != t.City && gap >= 0 && gap < time.Hour {
		return Hit{"impossible-travel", 60, fmt.Sprintf("%s -> %s in %s", prev.City, t.City, gap.Round(time.Second))}, true
	}
	return Hit{}, false
}

// AmountSpikeRule: amount is more than 10x the card's average.
// For a card with too little history (<3 txns), it falls back to a fixed limit of ₹50,000.
func AmountSpikeRule(t Transaction, past []Transaction) (Hit, bool) {
	if len(past) < 3 {
		if t.Amount > 50000 {
			return Hit{"amount-spike", 50, fmt.Sprintf("₹%.0f on a new card", t.Amount)}, true
		}
		return Hit{}, false
	}
	sum := 0.0
	for _, p := range past {
		sum += p.Amount
	}
	avg := sum / float64(len(past))
	if t.Amount > 10*avg {
		return Hit{"amount-spike", 50, fmt.Sprintf("%.0fx usual (avg ₹%.0f)", t.Amount/avg, avg)}, true
	}
	return Hit{}, false
}

var riskyCategories = map[string]bool{"crypto": true, "giftcard": true, "gambling": true}

// RiskyMerchantRule: merchant category is often used for fraud.
func RiskyMerchantRule(t Transaction, _ []Transaction) (Hit, bool) {
	if riskyCategories[t.Category] {
		return Hit{"risky-merchant", 40,t.Category + " merchant"}, true
	}
	return Hit{}, false
}

// NightSpendRule: more than ₹20,000 between midnight and 5 am.
func NightSpendRule(t Transaction, _ []Transaction) (Hit, bool) {
	if h := t.Time.Hour(); h < 5 && t.Amount > 20000 {
		return Hit{"night-spend", 40,fmt.Sprintf("₹%.0f at %s", t.Amount, t.Time.Format("15:04"))}, true
	}
	return Hit{}, false
}
