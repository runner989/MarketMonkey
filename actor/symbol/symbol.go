package symbol

import (
	"marketmonkey/actor/orderbook"
	"marketmonkey/actor/publish"
	"marketmonkey/actor/stat"
	"marketmonkey/actor/trade"
	"marketmonkey/event"

	"github.com/anthdm/hollywood/actor"
)

type Symbol struct {
	pair       event.Pair
	statPID    *actor.PID
	bookPID    *actor.PID
	publishPID *actor.PID
	tradePID   *actor.PID
}

func New(pair event.Pair) actor.Producer {
	return func() actor.Receiver {
		return &Symbol{
			pair: pair,
		}
	}
}

func (s *Symbol) Receive(c *actor.Context) {
	switch c.Message().(type) {
	case actor.Started:
		s.start(c)
	case event.Trade:
		c.Forward(s.bookPID)
		c.Forward(s.tradePID)
	case event.Stat:
		c.Forward(s.statPID)
	case event.BookUpdate:
		c.Forward(s.bookPID)
	}
}

func (s *Symbol) start(c *actor.Context) {
	s.statPID = c.SpawnChild(stat.New(s.pair), "stat", actor.WithID(s.pair.Symbol))
	s.bookPID = c.SpawnChild(orderbook.New(s.pair), "book", actor.WithID(s.pair.Symbol))
	s.tradePID = c.SpawnChild(trade.New(s.pair), "trade", actor.WithID(s.pair.Symbol))
	s.publishPID = c.SpawnChild(publish.New(s.pair), "publish", actor.WithID(s.pair.Symbol))
}
