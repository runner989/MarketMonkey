package trade

import (
	"math"

	"marketmonkey/event"
	"marketmonkey/settings"

	"github.com/anthdm/hollywood/actor"
)

type Trade struct {
	pair       event.Pair
	publishPID *actor.PID
	samplers   map[int64]*CandleSampler
	lastUnix   int64
	lastPrice  float64
	ctx        *actor.Context
}

func New(pair event.Pair) actor.Producer {
	return func() actor.Receiver {
		return &Trade{
			pair:     pair,
			samplers: make(map[int64]*CandleSampler),
		}
	}
}

func (t *Trade) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		t.ctx = c
		for _, tf := range settings.TickIntervals {
			if !tf.Disabled {
				t.samplers[tf.Interval] = NewCandleSampler(tf.Interval, t.onCandle)
			}
		}
		t.publishPID = c.Parent().Child("publish/" + t.pair.Symbol)
	case event.Trade:
		if msg.Unix > t.lastUnix || t.lastPrice == 0 {
			t.lastUnix = msg.Unix / 1000
			t.lastPrice = msg.Price
		}
		c.Forward(t.publishPID)
		for _, sampler := range t.samplers {
			sampler.ProcessTrades([]event.Trade{msg})
		}
	}
}

func (t *Trade) onCandle(candle event.Candle) {
	t.ctx.Send(t.publishPID, candle)
}

type CandleSampler struct {
	timeframe  int64
	candle     *event.Candle
	handleFunc func(event.Candle)
}

func NewCandleSampler(timeframe int64, fn func(c event.Candle)) *CandleSampler {
	return &CandleSampler{
		timeframe:  timeframe,
		candle:     &event.Candle{},
		handleFunc: fn,
	}
}

func (s *CandleSampler) ProcessTrades(trades []event.Trade) {
	for _, trade := range trades {
		unix := trade.Unix / 1000 / s.timeframe * s.timeframe
		if s.candle.Unix > 0 && s.candle.Unix+s.timeframe <= unix {
			s.candle = &event.Candle{
				Open:      trade.Price,
				Timeframe: s.timeframe,
			}
		}
		if s.candle.Unix == 0 {
			s.candle.Unix = unix
			s.candle.Timeframe = s.timeframe
		}
		s.candle.Close = trade.Price
		s.candle.High = math.Max(trade.Price, s.candle.High)
		s.candle.Low = math.Min(trade.Price, s.candle.Low)
		if s.candle.Open == 0 {
			s.candle.Open = trade.Price
		}
		if s.candle.Low == 0 {
			s.candle.Low = trade.Price
		}
		// TODO: round this
		if trade.IsBuy {
			s.candle.Vbuy = s.candle.Vbuy + trade.Qty
			s.candle.Tbuy++
		} else {
			s.candle.Vsell = s.candle.Vsell + trade.Qty
			s.candle.Tsell++
		}

		candle := event.Candle{
			Pair:      trade.Pair,
			Open:      s.candle.Open,
			Close:     s.candle.Close,
			High:      s.candle.High,
			Low:       s.candle.Low,
			Timeframe: s.candle.Timeframe,
			Vbuy:      s.candle.Vbuy,
			Vsell:     s.candle.Vsell,
			Unix:      s.candle.Unix,
			Tbuy:      s.candle.Tbuy,
			Tsell:     s.candle.Tsell,
		}

		s.handleFunc(candle)
	}
}
