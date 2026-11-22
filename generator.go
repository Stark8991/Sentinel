package main

import (
	"fmt"
	"math/rand"
	"time"
)

// cardProfile describes how a card normally behaves.
type cardProfile struct {
	ID          string
	HomeCity    string
	NormalSpend float64
}

type merchant struct {
	Name     string
	Category string
}

var (
	homeCities = []string{"Chennai", "Mumbai", "Bengaluru", "Delhi", "Hyderabad", "Pune", "Kolkata"}
	farCities  = []string{"Dubai", "London", "Singapore", "New York"}

	normalMerchants = []merchant{
		{"Swiggy", "food"}, {"Zomato", "food"}, {"Amazon", "shopping"}, {"Flipkart", "shopping"},
		{"BigBazaar", "grocery"}, {"Uber", "travel"}, {"IndianOil", "fuel"}, {"Netflix", "subscription"},
	}
	riskyMerchants = []merchant{{"CryptoX", "crypto"}, {"GiftHub", "giftcard"}, {"BetKing", "gambling"}}
)

// Generator produces an endless stream of fake transactions with some planted fraud.
type Generator struct {
	cards   []cardProfile
	seq     int
	rng     *rand.Rand
	pending []Transaction // extra transactions queued by multi-step fraud patterns
}

func NewGenerator(numCards int) *Generator {
	g := &Generator{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	for i := 0; i < numCards; i++ {
		g.cards = append(g.cards, cardProfile{
			ID:          fmt.Sprintf("CARD-%04d", 1000+i), // unique per card
			HomeCity:    homeCities[g.rng.Intn(len(homeCities))],
			NormalSpend: float64(200 + g.rng.Intn(2800)), // ₹200 – ₹3000
		})
	}
	return g
}

// Run sends transactions to out at roughly perSecond, until the program stops.
func (g *Generator) Run(out chan<- Transaction, perSecond int) {
	ticker := time.NewTicker(time.Second / time.Duration(perSecond))
	defer ticker.Stop()
	for range ticker.C {
		out <- g.next()
	}
}

func (g *Generator) next() Transaction {
	if len(g.pending) > 0 {
		t := g.pending[0]
		g.pending = g.pending[1:]
		t.Time = time.Now()
		return t
	}
	card := g.cards[g.rng.Intn(len(g.cards))]
	if g.rng.Float64() < 0.05 { // ~5% of the time, plant a fraud pattern
		return g.fraud(card)
	}
	return g.normal(card)
}

func (g *Generator) newTxn(card cardProfile, amount float64, m merchant, city, planted string) Transaction {
	g.seq++
	return Transaction{
		ID: fmt.Sprintf("TXN-%06d", g.seq), CardID: card.ID, Amount: amount,
		Merchant: m.Name, Category: m.Category, City: city, Time: time.Now(), Planted: planted,
	}
}

// normal: an amount near the card's usual spend, in its home city, at an ordinary merchant.
func (g *Generator) normal(card cardProfile) Transaction {
	amount := card.NormalSpend * (0.5 + g.rng.Float64()) // 50%–150% of normal
	m := normalMerchants[g.rng.Intn(len(normalMerchants))]
	return g.newTxn(card, amount, m, card.HomeCity, "")
}

func (g *Generator) fraud(card cardProfile) Transaction {
	switch g.rng.Intn(5) {
	case 0: // burst: 6 quick transactions on the same card.
		// Only the 5th and 6th are tagged: a single payment can't look suspicious until the burst builds up.
		first := g.normal(card)
		for i := 2; i <= 6; i++ {
			t := g.normal(card)
			if i >= 5 {
				t.Planted = "burst"
			}
			g.pending = append(g.pending, t)
		}
		return first
	case 1: // city jump: home city, then a far-away city moments later
		home := g.normal(card)
		far := g.newTxn(card, card.NormalSpend, normalMerchants[g.rng.Intn(len(normalMerchants))],
			farCities[g.rng.Intn(len(farCities))], "city-jump")
		g.pending = append(g.pending, far)
		return home
	case 2: // huge amount: 20–40x the normal spend
		m := normalMerchants[g.rng.Intn(len(normalMerchants))]
		return g.newTxn(card, card.NormalSpend*float64(20+g.rng.Intn(21)), m, card.HomeCity, "amount-spike")
	case 3: // risky merchant
		m := riskyMerchants[g.rng.Intn(len(riskyMerchants))]
		return g.newTxn(card, card.NormalSpend*(1+g.rng.Float64()*2), m, card.HomeCity, "risky-merchant")
	default: // odd hour: large spend stamped at ~3 am
		m := normalMerchants[g.rng.Intn(len(normalMerchants))]
		t := g.newTxn(card, 30000+float64(g.rng.Intn(40000)), m, card.HomeCity, "night-spend")
		now := time.Now()
		t.Time = time.Date(now.Year(), now.Month(), now.Day(), 3, g.rng.Intn(60), 0, 0, now.Location())
		return t
	}
}
