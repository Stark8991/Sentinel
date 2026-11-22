package main

import (
	"testing"
	"time"
)

// base is a fixed daytime moment so tests don't depend on the real clock.
var base = time.Date(2026, 11, 22, 14, 0, 0, 0, time.Local)

func txn(amount float64, category, city string, at time.Time) Transaction {
	return Transaction{CardID: "CARD-TEST", Amount: amount, Category: category, City: city, Time: at}
}

// history builds n past transactions of the given amount, `gap` apart, ending just before `end`.
func history(n int, amount float64, city string, end time.Time, gap time.Duration) []Transaction {
	var past []Transaction
	for i := n; i >= 1; i-- {
		past = append(past, txn(amount, "food", city, end.Add(-time.Duration(i)*gap)))
	}
	return past
}

func TestVelocityRule(t *testing.T) {
	cases := []struct {
		name string
		past []Transaction
		want bool
	}{
		{"5 txns within 60s fires", history(4, 500, "Chennai", base, 10*time.Second), true},
		{"only 4 txns does not fire", history(3, 500, "Chennai", base, 10*time.Second), false},
		{"5 txns spread over 10 minutes does not fire", history(4, 500, "Chennai", base, 2*time.Minute), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, got := VelocityRule(txn(500, "food", "Chennai", base), c.past); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestImpossibleTravelRule(t *testing.T) {
	cases := []struct {
		name string
		past []Transaction
		city string
		want bool
	}{
		{"Chennai then Dubai 5 min later fires", []Transaction{txn(500, "food", "Chennai", base.Add(-5 * time.Minute))}, "Dubai", true},
		{"Chennai then Dubai 3 hours later does not fire", []Transaction{txn(500, "food", "Chennai", base.Add(-3 * time.Hour))}, "Dubai", false},
		{"same city does not fire", []Transaction{txn(500, "food", "Chennai", base.Add(-1 * time.Minute))}, "Chennai", false},
		{"no history does not fire", nil, "Dubai", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, got := ImpossibleTravelRule(txn(500, "food", c.city, base), c.past); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestAmountSpikeRule(t *testing.T) {
	usual := history(5, 500, "Chennai", base, time.Hour)
	cases := []struct {
		name   string
		past   []Transaction
		amount float64
		want   bool
	}{
		{"usually ₹500, now ₹10,000 fires", usual, 10000, true},
		{"usually ₹500, now ₹1,500 does not fire", usual, 1500, false},
		{"new card ₹60,000 fires", nil, 60000, true},
		{"new card ₹5,000 does not fire", nil, 5000, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, got := AmountSpikeRule(txn(c.amount, "shopping", "Chennai", base), c.past); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestRiskyMerchantRule(t *testing.T) {
	cases := map[string]bool{"crypto": true, "giftcard": true, "gambling": true, "food": false, "shopping": false}
	for category, want := range cases {
		t.Run(category, func(t *testing.T) {
			if _, got := RiskyMerchantRule(txn(1000, category, "Chennai", base), nil); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestNightSpendRule(t *testing.T) {
	at3am := time.Date(2026, 11, 22, 3, 0, 0, 0, time.Local)
	cases := []struct {
		name   string
		amount float64
		at     time.Time
		want   bool
	}{
		{"₹30,000 at 3 am fires", 30000, at3am, true},
		{"₹30,000 at 3 pm does not fire", 30000, at3am.Add(12 * time.Hour), false},
		{"₹500 at 3 am does not fire", 500, at3am, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, got := NightSpendRule(txn(c.amount, "shopping", "Chennai", c.at), nil); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
