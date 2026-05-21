package matcher

// Side is the direction of an order: a Buy wants to acquire the asset,
// a Sell wants to dispose of it. A match always pairs one of each.
type Side uint8

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

// Order is a resting or incoming limit order.
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
	Price    int64
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
