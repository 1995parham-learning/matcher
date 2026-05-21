package matcher

// Side is the direction of an order: a Buy wants to acquire the asset,
// a Sell wants to dispose of it. A match always pairs one of each.
type Side uint8

// Buy and Sell are the two sides of every order and trade.
const (
	Buy Side = iota
	Sell
)

func (s Side) String() string {
	if s == Buy {
		return "BUY"
	}
	return "SELL"
}

// Type distinguishes how aggressively an order is willing to trade.
//
//   - Limit:  trades only at Price or better. Unfilled residual rests
//     on the book.
//   - Market: trades at any price the opposite side is offering.
//     Price is ignored. Unfilled residual is cancelled, not rested —
//     a market order has no price to rest at.
//
// Limit is the zero value, so existing limit-only callers keep
// working without naming the field.
type Type uint8

// Limit and Market are the supported order types. See Type for
// behavioural differences.
const (
	Limit Type = iota
	Market
)

func (t Type) String() string {
	if t == Market {
		return "MKT"
	}
	return "LMT"
}

// Order is a resting or incoming order.
//
// Price and Quantity are integers to keep the matcher exact — real
// engines never use floats. Pick a unit (cents, ticks, satoshis) and
// stick with it.
//
// Seq is a monotonically increasing sequence number assigned by the
// book when the order is accepted. It is the basis for time priority
// (lower Seq = earlier = matched first at the same price).
type Order struct {
	ID       uint64
	Side     Side
	Type     Type
	Price    int64 // ignored when Type == Market
	Quantity int64
	Seq      uint64
}

// Trade is the record of a single match between a buy and a sell.
// One incoming order may produce many Trades if it sweeps multiple
// resting orders or price levels.
type Trade struct {
	BuyOrderID  uint64
	SellOrderID uint64
	Price       int64 // always the price of the *resting* (maker) order
	Quantity    int64
}
