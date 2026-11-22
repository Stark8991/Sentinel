package main

import "sync"

const historySize = 20

// History keeps each card's most recent transactions. Safe for use from multiple goroutines.
type History struct {
	mu    sync.RWMutex
	cards map[string][]Transaction
}

func NewHistory() *History {
	return &History{cards: make(map[string][]Transaction)}
}

// Get returns a copy of the card's recent transactions (oldest first).
func (h *History) Get(cardID string) []Transaction {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]Transaction(nil), h.cards[cardID]...)
}

// Add records a transaction, keeping only the last historySize per card.
func (h *History) Add(t Transaction) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := append(h.cards[t.CardID], t)
	if len(list) > historySize {
		list = list[len(list)-historySize:]
	}
	h.cards[t.CardID] = list
}
