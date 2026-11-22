package main

import (
	"fmt"
	"time"
)

// Transaction is a single card payment flowing through the engine.
type Transaction struct {
	ID       string
	CardID   string
	Amount   float64 // in rupees
	Merchant string
	Category string // food, shopping, travel, fuel, crypto, giftcard ...
	City     string
	Time     time.Time
	Planted  string // which fraud pattern the generator planted ("" = normal); used only to check the engine
}

func (t Transaction) String() string {
	return fmt.Sprintf("%s  %s  ₹%10.2f  %-10s %-12s %-10s %s",
		t.ID, t.CardID, t.Amount, t.Merchant, t.Category, t.City, t.Time.Format("15:04:05"))
}
