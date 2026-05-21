package main

import (
	"fmt"

	"github.com/parham/matcher"
)

// A small narrated walk-through of what the engine does. Run with:
//
//	go run ./cmd/demo
func main() {
	b := matcher.NewBook()

	seed := []matcher.Order{
		{ID: 1, Side: matcher.Sell, Price: 102, Quantity: 5},
		{ID: 2, Side: matcher.Sell, Price: 101, Quantity: 3},
		{ID: 3, Side: matcher.Sell, Price: 101, Quantity: 2},
		{ID: 4, Side: matcher.Buy, Price: 99, Quantity: 4},
		{ID: 5, Side: matcher.Buy, Price: 98, Quantity: 6},
	}
	for _, o := range seed {
		b.Match(o)
	}

	fmt.Println("Initial book:")
	fmt.Println(b)

	// An aggressive buy that crosses both ask levels at 101 (5 total)
	// and partially fills 102. Residual rests on the bid side.
	incoming := matcher.Order{ID: 6, Side: matcher.Buy, Price: 102, Quantity: 7}
	fmt.Printf("Incoming: %s %d @ %d (id=%d)\n\n", incoming.Side, incoming.Quantity, incoming.Price, incoming.ID)

	trades, rest := b.Match(incoming)
	for _, t := range trades {
		fmt.Printf("  TRADE  buy=%d sell=%d  %d @ %d\n", t.BuyOrderID, t.SellOrderID, t.Quantity, t.Price)
	}
	if rest != nil {
		fmt.Printf("  RESTING residual: id=%d %s %d @ %d\n", rest.ID, rest.Side, rest.Quantity, rest.Price)
	}

	fmt.Println("\nBook after match:")
	fmt.Println(b)
}
