// Package matcher is a small, single-instrument order book and
// matching engine used as a study aid for trading-system concepts:
// price-time priority, partial fills, limit and market orders.
package matcher

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// priceLevel is a FIFO queue of orders resting at one price.
// Time priority within a level is preserved by always appending new
// orders to the tail and matching from the head.
type priceLevel struct {
	price  int64
	orders []*Order
}

func (pl *priceLevel) totalQty() int64 {
	var n int64
	for _, o := range pl.orders {
		n += o.Quantity
	}
	return n
}

// Book is a single-instrument limit order book.
//
// Bids and Asks are kept sorted by *price priority*:
//
//	bids: highest price first  (best bid at index 0)
//	asks: lowest  price first  (best ask at index 0)
//
// Within a level, orders sit in FIFO order. Together this gives the
// classic "price-time priority" matching model used by most venues.
type Book struct {
	Bids []*priceLevel
	Asks []*priceLevel

	nextSeq uint64
}

// NewBook returns an empty order book.
func NewBook() *Book { return &Book{} }

// BestBid returns the highest bid price and ok=true if a bid exists.
func (b *Book) BestBid() (price int64, ok bool) {
	if len(b.Bids) == 0 {
		return 0, false
	}
	return b.Bids[0].price, true
}

// BestAsk returns the lowest ask price and ok=true if an ask exists.
func (b *Book) BestAsk() (price int64, ok bool) {
	if len(b.Asks) == 0 {
		return 0, false
	}
	return b.Asks[0].price, true
}

// Match is the heart of the engine.
//
// It takes an incoming order and walks the opposite side of the book
// from best price outward, generating trades for as long as:
//   - the incoming order still has quantity left, AND
//   - the best resting price still "crosses" the incoming order.
//
// Crossing depends on order type:
//   - Limit:  a buy crosses when its limit >= the best ask; a sell
//     crosses when its limit <= the best bid.
//   - Market: always crosses — keep matching until the book runs out
//     of liquidity on that side.
//
// What happens to an unfilled residual also depends on type. A Limit
// residual is added to the book and returned via rest. A Market
// residual is dropped on the floor — there is no price at which it
// could rest — and rest is nil.
//
// Trades are reported at the *resting* order's price. The order that
// was already on the book is the maker; the incoming order is the
// taker. The taker pays whatever price the maker was offering.
func (b *Book) Match(incoming Order) (trades []Trade, rest *Order) {
	if incoming.Quantity <= 0 {
		return nil, nil
	}

	opposite, crosses := b.oppositeSide(incoming)

	for incoming.Quantity > 0 && len(*opposite) > 0 {
		best := (*opposite)[0]
		if !crosses(best.price) {
			break
		}

		// Walk the FIFO at this price level. Each iteration either
		// consumes the head order entirely or partially fills it and
		// exhausts the incoming order.
		for incoming.Quantity > 0 && len(best.orders) > 0 {
			head := best.orders[0]

			qty := min(head.Quantity, incoming.Quantity)

			t := Trade{Price: best.price, Quantity: qty}
			if incoming.Side == Buy {
				t.BuyOrderID, t.SellOrderID = incoming.ID, head.ID
			} else {
				t.BuyOrderID, t.SellOrderID = head.ID, incoming.ID
			}
			trades = append(trades, t)

			head.Quantity -= qty
			incoming.Quantity -= qty

			if head.Quantity == 0 {
				best.orders = best.orders[1:]
			}
		}

		if len(best.orders) == 0 {
			*opposite = (*opposite)[1:]
		}
	}

	if incoming.Quantity > 0 && incoming.Type == Limit {
		rest = b.rest(incoming)
	}
	return trades, rest
}

// String renders a compact, human-readable snapshot of the book —
// useful for tinkering at the REPL or eyeballing test failures.
func (b *Book) String() string {
	var sb strings.Builder
	sb.WriteString("            ASKS\n")
	// Print asks from worst (highest) to best (lowest) so the spread
	// sits in the middle, like a ladder display on a trading screen.
	for _, pl := range slices.Backward(b.Asks) {
		fmt.Fprintf(&sb, "  %6d  x %d\n", pl.price, pl.totalQty())
	}
	sb.WriteString("  ------- spread -------\n")
	for _, pl := range b.Bids {
		fmt.Fprintf(&sb, "  %6d  x %d\n", pl.price, pl.totalQty())
	}
	sb.WriteString("            BIDS\n")
	return sb.String()
}

// oppositeSide returns a pointer to the slice the incoming order will
// match against, plus the "crosses" predicate for that side.
//
// Returning the pointer (not the slice) lets Match mutate the book —
// reslicing as price levels are exhausted — without re-finding the
// side every iteration. Capturing the incoming order's limit (or its
// Market type) inside the closure keeps the hot inner loop branch-free.
func (b *Book) oppositeSide(o Order) (*[]*priceLevel, func(restingPrice int64) bool) {
	always := func(int64) bool { return true }
	if o.Side == Buy {
		if o.Type == Market {
			return &b.Asks, always
		}
		return &b.Asks, func(resting int64) bool { return o.Price >= resting }
	}
	if o.Type == Market {
		return &b.Bids, always
	}
	return &b.Bids, func(resting int64) bool { return o.Price <= resting }
}

// rest inserts the residual of a partially filled (or untouched)
// incoming order into the correct side of the book, assigning it a
// sequence number for time priority.
func (b *Book) rest(o Order) *Order {
	b.nextSeq++
	o.Seq = b.nextSeq
	stored := &o

	var levels *[]*priceLevel
	// `better` answers: does price `a` have priority over price `b`?
	// For bids: higher is better. For asks: lower is better.
	var better func(a, b int64) bool
	if o.Side == Buy {
		levels = &b.Bids
		better = func(a, b int64) bool { return a > b }
	} else {
		levels = &b.Asks
		better = func(a, b int64) bool { return a < b }
	}

	// Find the first level whose price is *not* better than ours; that
	// is where we either land (if prices match) or insert a new level.
	i := sort.Search(len(*levels), func(i int) bool {
		return !better((*levels)[i].price, o.Price)
	})

	if i < len(*levels) && (*levels)[i].price == o.Price {
		(*levels)[i].orders = append((*levels)[i].orders, stored)
		return stored
	}

	*levels = append(*levels, nil)
	copy((*levels)[i+1:], (*levels)[i:])
	(*levels)[i] = &priceLevel{price: o.Price, orders: []*Order{stored}}
	return stored
}
