package matcher

import (
	"reflect"
	"testing"
)

// Each test isolates one matching concept so the file doubles as a
// readable tour of what the engine does.

func TestNoCross_RestsOnBook(t *testing.T) {
	// A buy at 99 cannot cross an ask at 101 — it just rests.
	b := NewBook()
	b.Match(Order{ID: 1, Side: Sell, Price: 101, Quantity: 10})

	trades, rest := b.Match(Order{ID: 2, Side: Buy, Price: 99, Quantity: 5})
	if len(trades) != 0 {
		t.Fatalf("expected no trades, got %v", trades)
	}
	if rest == nil || rest.Quantity != 5 {
		t.Fatalf("expected residual of 5 to rest, got %+v", rest)
	}
	if got, _ := b.BestBid(); got != 99 {
		t.Fatalf("best bid = %d, want 99", got)
	}
}

func TestFullCross_OneMaker(t *testing.T) {
	// Incoming buy at 101 fully consumes a resting ask at 100.
	b := NewBook()
	b.Match(Order{ID: 1, Side: Sell, Price: 100, Quantity: 10})

	trades, rest := b.Match(Order{ID: 2, Side: Buy, Price: 101, Quantity: 10})
	want := []Trade{{BuyOrderID: 2, SellOrderID: 1, Price: 100, Quantity: 10}}
	if !reflect.DeepEqual(trades, want) {
		t.Fatalf("trades = %+v, want %+v", trades, want)
	}
	if rest != nil {
		t.Fatalf("expected no residual, got %+v", rest)
	}
	if _, ok := b.BestAsk(); ok {
		t.Fatalf("ask side should be empty")
	}
}

func TestPartialFill_ResidualRestsAtTakerPrice(t *testing.T) {
	// Resting ask of 6 @ 100; incoming buy of 10 @ 100.
	// 6 matches, residual 4 rests as a *bid* at 100.
	b := NewBook()
	b.Match(Order{ID: 1, Side: Sell, Price: 100, Quantity: 6})

	trades, rest := b.Match(Order{ID: 2, Side: Buy, Price: 100, Quantity: 10})
	if len(trades) != 1 || trades[0].Quantity != 6 {
		t.Fatalf("expected one trade of 6, got %+v", trades)
	}
	if rest == nil || rest.Quantity != 4 || rest.Side != Buy {
		t.Fatalf("expected residual buy of 4, got %+v", rest)
	}
	if got, _ := b.BestBid(); got != 100 {
		t.Fatalf("best bid = %d, want 100", got)
	}
}

func TestTimePriority_FIFO(t *testing.T) {
	// Two asks at the same price. The earlier one fills first.
	b := NewBook()
	b.Match(Order{ID: 1, Side: Sell, Price: 100, Quantity: 5})
	b.Match(Order{ID: 2, Side: Sell, Price: 100, Quantity: 5})

	trades, _ := b.Match(Order{ID: 3, Side: Buy, Price: 100, Quantity: 7})
	if len(trades) != 2 {
		t.Fatalf("want 2 trades, got %d", len(trades))
	}
	if trades[0].SellOrderID != 1 || trades[1].SellOrderID != 2 {
		t.Fatalf("FIFO violated: %+v", trades)
	}
	if trades[0].Quantity != 5 || trades[1].Quantity != 2 {
		t.Fatalf("unexpected quantities: %+v", trades)
	}
}

func TestPricePriority_SweepLevels(t *testing.T) {
	// Aggressive buy sweeps two ask levels until its limit is reached.
	b := NewBook()
	b.Match(Order{ID: 1, Side: Sell, Price: 101, Quantity: 3})
	b.Match(Order{ID: 2, Side: Sell, Price: 102, Quantity: 3})
	b.Match(Order{ID: 3, Side: Sell, Price: 103, Quantity: 3}) // out of reach

	trades, rest := b.Match(Order{ID: 4, Side: Buy, Price: 102, Quantity: 10})
	if len(trades) != 2 {
		t.Fatalf("want 2 trades (across 2 levels), got %+v", trades)
	}
	if trades[0].Price != 101 || trades[1].Price != 102 {
		t.Fatalf("expected best-price-first, got %+v", trades)
	}
	// 10 wanted, 6 filled, 4 residual rests as a bid at 102.
	if rest == nil || rest.Quantity != 4 {
		t.Fatalf("expected residual of 4, got %+v", rest)
	}
	// The 103 ask must be untouched — it never crossed.
	if got, _ := b.BestAsk(); got != 103 {
		t.Fatalf("best ask = %d, want 103", got)
	}
}

func TestMakerPriceWins_TakerImproves(t *testing.T) {
	// A buyer willing to pay 105 hits a resting ask at 100. The trade
	// prints at 100 (the maker's price) — taker gets price improvement.
	b := NewBook()
	b.Match(Order{ID: 1, Side: Sell, Price: 100, Quantity: 1})

	trades, _ := b.Match(Order{ID: 2, Side: Buy, Price: 105, Quantity: 1})
	if len(trades) != 1 || trades[0].Price != 100 {
		t.Fatalf("expected trade @ maker price 100, got %+v", trades)
	}
}

func TestSellSideMirrorsBuySide(t *testing.T) {
	// A symmetric scenario: aggressive sell sweeps bids.
	b := NewBook()
	b.Match(Order{ID: 1, Side: Buy, Price: 100, Quantity: 4})
	b.Match(Order{ID: 2, Side: Buy, Price: 99, Quantity: 4})

	trades, rest := b.Match(Order{ID: 3, Side: Sell, Price: 99, Quantity: 10})
	if len(trades) != 2 || trades[0].Price != 100 || trades[1].Price != 99 {
		t.Fatalf("expected sweep 100 then 99, got %+v", trades)
	}
	if rest == nil || rest.Quantity != 2 || rest.Side != Sell {
		t.Fatalf("expected residual sell of 2 @ 99, got %+v", rest)
	}
}
