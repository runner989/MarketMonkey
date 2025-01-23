package stat

import (
	"fmt"
	"marketmonkey/event"

	"github.com/anthdm/hollywood/actor"
)

type Stat struct {
	pair event.Pair
}

func New(pair event.Pair) actor.Producer {
	return func() actor.Receiver {
		return &Stat{
			pair: pair,
		}
	}
}

func (s *Stat) Receive(c *actor.Context) {
	switch v := c.Message().(type) {
	case actor.Started:
		fmt.Printf("stat started %v\n", s.pair)
	case event.Stat:
		fmt.Printf("stat %v\n", v)
	}
}
