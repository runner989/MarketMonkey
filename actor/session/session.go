package session

import (
	act "marketmonkey/actor"
	"marketmonkey/actor/publish"
	"marketmonkey/event"

	"github.com/anthdm/hollywood/actor"
)

type Stream struct {
	Stream    event.Stream
	Timeframe int64
}

type Session struct {
	pair       event.Pair
	eventCh    chan any
	streams    []Stream
	publishPID *actor.PID
}

func New(eventCh chan any, pair event.Pair, streams []Stream) actor.Producer {
	return func() actor.Receiver {
		return &Session{
			pair:       pair,
			eventCh:    eventCh,
			streams:    streams,
			publishPID: act.GetPublishPID(pair),
		}
	}
}

func (s *Session) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		keys := make([]uint32, len(s.streams))
		for i := 0; i < len(s.streams); i++ {
			stream := s.streams[i]
			keys[i] = publish.CreateRouteKey(s.pair, stream.Stream, stream.Timeframe)
		}
		c.Send(s.publishPID, event.PubSub{Streams: keys})
	case actor.Stopped:
		keys := make([]uint32, len(s.streams))
		for i := 0; i < len(s.streams); i++ {
			stream := s.streams[i]
			keys[i] = publish.CreateRouteKey(s.pair, stream.Stream, stream.Timeframe)
		}
		c.Send(s.publishPID, event.PubUnsub{Streams: keys})
		close(s.eventCh)
	case event.Orderbook, event.Trade, event.Heatmap, event.Candle:
		s.eventCh <- msg
	}
}
