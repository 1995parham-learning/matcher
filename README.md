# matcher

[![ci](https://github.com/1995parham-learning/matcher/actions/workflows/ci.yml/badge.svg)](https://github.com/1995parham-learning/matcher/actions/workflows/ci.yml)

A small Go order book and matching engine, written to make the
mechanics of a price-time-priority exchange easy to read. No
dependencies — just the standard library.

## What it demonstrates

- **Price-time priority.** Bids are kept sorted descending, asks
  ascending; orders inside a price level form a FIFO queue.
- **Crossing.** A buy crosses when its limit is at or above the best
  ask; a sell crosses when its limit is at or below the best bid.
- **Partial fills.** An incoming order keeps matching until either its
  quantity is exhausted or the next resting level no longer crosses.
  Any residual rests on the book (limit orders) or is cancelled
  (market orders).
- **Maker price wins.** Trades print at the resting order's price, so
  an aggressive taker can get price improvement.
- **Limit vs market orders.** Market orders ignore price and never
  rest — unfilled quantity is dropped.

## Quick start

```bash
go run ./cmd/demo
```

```
Initial book:
            ASKS
     102  x 5
     101  x 5
  ------- spread -------
      99  x 4
      98  x 6
            BIDS

Incoming: BUY 7 @ 102 (id=6)

  TRADE  buy=6 sell=2  3 @ 101
  TRADE  buy=6 sell=3  2 @ 101
  TRADE  buy=6 sell=1  2 @ 102

Book after match:
            ASKS
     102  x 3
  ------- spread -------
      99  x 4
      98  x 6
            BIDS
```

## Using the package

```go
import "github.com/1995parham-learning/matcher"

b := matcher.NewBook()

// Seed a resting ask.
b.Match(matcher.Order{ID: 1, Side: matcher.Sell, Price: 100, Quantity: 10})

// Send a crossing limit buy.
trades, rest := b.Match(matcher.Order{
    ID: 2, Side: matcher.Buy, Price: 101, Quantity: 6,
})

// trades = [{BuyOrderID:2, SellOrderID:1, Price:100, Quantity:6}]
// rest   = nil (fully filled)
// The remaining 4 of order 1 still rests at 100.
```

A market order is the same call with `Type: matcher.Market` and any
`Price` value (it is ignored). Unfilled market quantity is cancelled
instead of resting.

## Layout

```
order.go            Side, Type, Order, Trade
book.go             Book + Match (the matching loop)
book_test.go        One test per concept
cmd/demo/main.go    Narrated walk-through
```

## Run the tests

```bash
go test -race ./...
golangci-lint run ./...
```

CI runs both on every push and pull request — see the badge above.

## Ideas to extend

The codebase is set up so adding more matching concepts is a small
change in one place:

- IOC / FOK — skip the `rest` call, or pre-check fillable quantity.
- Post-only — reject the order if it would cross.
- Self-trade prevention — skip a maker that shares an account with the
  taker.
- Pro-rata allocation — replace the FIFO inner loop.
- Stop / stop-limit — keep a separate trigger list keyed off last
  trade price.
- Order cancel / modify by ID — index orders so they can be located
  without walking the book.
