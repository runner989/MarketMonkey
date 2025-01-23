package app

import (
	"marketmonkey/actor/session"
	"marketmonkey/event"
	"marketmonkey/settings"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type LineChartLayer struct {
	pair       event.Pair
	sessionPID *actor.PID
	eventCh    chan any
	trades     []event.Candle
	lastUnix   int64
	lastPrice  float64
	interval   int64
}

func NewLineChartLayer(pair event.Pair) *LineChartLayer {
	return &LineChartLayer{
		pair:     pair,
		eventCh:  make(chan any),
		trades:   []event.Candle{},
		lastUnix: time.Now().Unix(),
	}
}

func (l *LineChartLayer) initialize(chart *ChartWidget) {
	l.interval = chart.interval
	streams := []session.Stream{{
		Stream:    event.StreamCandles,
		Timeframe: l.interval,
	}}

	l.sessionPID = app.engine.Spawn(session.New(l.eventCh, l.pair, streams), "session")

	go l.receiveData()
}

func (l *LineChartLayer) update(chart *ChartWidget) {
	chart.lastPrice = l.lastPrice
}

func (l *LineChartLayer) render(screen *ebiten.Image, chart *ChartWidget) {
	rect := chart.GetWidget().Rect
	minY := float32(rect.Min.Y)
	maxY := float32(rect.Max.Y)
	minX := float32(rect.Min.X)
	maxX := float32(rect.Max.X)

	for i := 0; i < len(l.trades)-1; i++ {
		t1 := l.trades[i]
		t2 := l.trades[i+1]

		barIndex1 := float64(i)
		barIndex2 := float64(i + 1)

		// Calculate screen coordinates
		x1 := minX + float32((barIndex1-chart.barOffset)*chart.barWidth)
		x2 := minX + float32((barIndex2-chart.barOffset)*chart.barWidth)
		y1 := chart.getPriceY(t1.Close)
		y2 := chart.getPriceY(t2.Close)

		// Skip if line is completely outside the visible area
		if (x1 < minX && x2 < minX) || (x1 > maxX && x2 > maxX) ||
			(y1 < minY && y2 < minY) || (y1 > maxY && y2 > maxY) {
			continue
		}

		// Clamp X coordinates
		if x1 < minX || x1 > maxX || x2 < minX || x2 > maxX {
			// Calculate slope
			if x2 != x1 { // Avoid division by zero
				slope := (y2 - y1) / (x2 - x1)

				// Clamp x1 if needed
				if x1 < minX {
					y1 = y1 + (minX-x1)*slope
					x1 = minX
				} else if x1 > maxX {
					y1 = y1 + (maxX-x1)*slope
					x1 = maxX
				}

				// Clamp x2 if needed
				if x2 < minX {
					y2 = y1 + (minX-x1)*slope
					x2 = minX
				} else if x2 > maxX {
					y2 = y1 + (maxX-x1)*slope
					x2 = maxX
				}
			}
		}

		// Clamp Y coordinates to visible bounds
		if y1 < minY {
			y1 = minY
		} else if y1 > maxY {
			y1 = maxY
		}
		if y2 < minY {
			y2 = minY
		} else if y2 > maxY {
			y2 = maxY
		}

		vector.StrokeLine(screen,
			x1,
			y1,
			x2,
			y2,
			float32(settings.LineChartStrokeWidth), settings.LineChartLineColor, true)
	}

	// Draw a line from the last bar to the current price
	if len(l.trades) > 0 {
		lastTrade := l.trades[len(l.trades)-1]
		lastBarIndex := float64(len(l.trades) - 1)
		currentBarIndex := lastBarIndex + 1 // One bar after the last trade

		// TODO: we might want to move this the chart widget on update
		// Calculate visible bar range
		visibleBars := float64(rect.Dx()) / chart.barWidth
		visibleStartBar := chart.barOffset
		visibleEndBar := visibleStartBar + visibleBars

		if currentBarIndex >= visibleStartBar && currentBarIndex <= visibleEndBar {
			x1 := minX + float32((lastBarIndex-chart.barOffset)*chart.barWidth)
			x2 := minX + float32((currentBarIndex-chart.barOffset)*chart.barWidth)
			y1 := chart.getPriceY(lastTrade.Close)
			y2 := chart.getPriceY(l.lastPrice)

			vector.StrokeLine(screen,
				x1,
				y1,
				x2,
				y2,
				float32(settings.LineChartStrokeWidth), settings.LineChartLineColor, true)
		}
	}
}

func (l *LineChartLayer) receiveData() {
	for ev := range l.eventCh {
		switch msg := ev.(type) {
		case event.Candle:
			l.lastPrice = msg.Close
			tradeUnix := msg.Unix
			if tradeUnix-l.lastUnix >= l.interval {
				l.trades = append(l.trades, msg)
				l.lastUnix = tradeUnix
			}
			if len(l.trades) < 2 {
				continue
			}
		}
	}
}

func (l *LineChartLayer) delete() {
	app.engine.Poison(l.sessionPID)
}
