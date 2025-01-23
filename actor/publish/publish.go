package publish

import (
	"fmt"
	"marketmonkey/event"

	"github.com/anthdm/hollywood/actor"
	"github.com/tidwall/murmur3"
)

type Publish struct {
	pair event.Pair

	subs map[uint32]map[*actor.PID]bool
	ctx  *actor.Context
}

func New(pair event.Pair) actor.Producer {
	return func() actor.Receiver {
		return &Publish{
			pair: pair,
			subs: make(map[uint32]map[*actor.PID]bool),
		}
	}
}

func (p *Publish) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		p.ctx = c
	case event.PubSub:
		for _, stream := range msg.Streams {
			sub, ok := p.subs[stream]
			if !ok {
				p.subs[stream] = make(map[*actor.PID]bool)
				p.subs[stream][c.Sender()] = true
			} else {
				sub[c.Sender()] = true
			}
			fmt.Printf("new subscription %s %d\n", c.Sender(), stream)
		}
	case event.PubUnsub:
		for _, stream := range msg.Streams {
			subs, ok := p.subs[stream]
			if ok {
				delete(subs, c.Sender())
				fmt.Printf("removed subscription %s %d\n", c.Sender(), stream)
			}
		}
	case event.Trade:
		p.broadcast(event.StreamTrades, msg)
	case event.Orderbook:
		p.broadcast(event.StreamOrderbook, msg)
	case event.Heatmap:
		p.broadcast(event.StreamHeatmap, msg)
	case event.Candle:
		p.broadcast(event.StreamCandles, msg)
	}
}

func (p *Publish) broadcast(stream event.Stream, msg event.TimeFramer) {
	key := CreateRouteKey(p.pair, stream, msg.GetTimeframe())
	subs, ok := p.subs[key]
	if ok {
		for pid := range subs {
			p.ctx.Send(pid, msg)
		}
	}
}

func CreateRouteKey(pair event.Pair, stream event.Stream, timeframe int64) uint32 {
	key := []byte(pair.Exchange)
	key = append(key, pair.Symbol...)
	key = append(key, byte(0xff&timeframe), byte(0xff&(timeframe>>8)), byte(0xff&(timeframe>>16)), byte(0xff&(timeframe>>24)))
	key = append(key, byte(0xff&stream), byte(0xff&(stream>>8)), byte(0xff&(stream>>16)), byte(0xff&(stream>>24)))
	return murmur3.Sum32Bytes(key)
}
