package app

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"time"

	"marketmonkey/actor/session"
	"marketmonkey/event"
	"marketmonkey/settings"

	"github.com/anthdm/hollywood/actor"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/colornames"
)

type BaseChartLayer struct {
	sessionPID *actor.PID
	eventCh    chan any
	candles    []event.Candle
	isDirty    bool
	image      *ebiten.Image
	triImage   *ebiten.Image
	visible    bool

	chart *ChartWidget
}

func NewBaseChartLayer(chart *ChartWidget) *BaseChartLayer {
	triImage := ebiten.NewImage(1, 1)
	triImage.Fill(colornames.White)

	l := &BaseChartLayer{
		eventCh:  make(chan any),
		candles:  []event.Candle{},
		triImage: triImage,
		visible:  true,
		chart:    chart,
	}
	l.chart = chart

	streams := []session.Stream{{
		Stream:    event.StreamCandles,
		Timeframe: chart.interval,
	}}

	l.chart.intervalChangeEvent.AddHandler(l.onIntervalChange)
	l.chart.chartTypeChangeEvent.AddHandler(l.onChartTypeChange)

	l.sessionPID = app.engine.Spawn(session.New(l.eventCh, l.chart.pair, streams), "session")

	go l.receiveData()

	return l
}

func (l *BaseChartLayer) update(chart *ChartWidget) {
	if l.image == nil {
		l.image = ebiten.NewImage(chart.GetWidget().Rect.Dx(), chart.GetWidget().Rect.Dy())
		l.image.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0})
	}
	if len(l.candles) == 0 {
		return
	}
	if chart.startTime.Unix() != l.candles[0].Unix {
		fmt.Println("chart start is not in sync with the first candle")
		chart.startTime = time.Unix(l.candles[0].Unix, 0)
	}
}

func (l *BaseChartLayer) renderUpdate(chart *ChartWidget) {
	if !l.visible {
		return
	}
	l.isDirty = false

	if chart.chartType == chartTypeCandles {
		l.renderVolumes(chart)
		l.renderCandlesticks(chart)
	}
	if chart.chartType == chartTypeLine {
		l.renderLines(chart)
	}
}

func (l *BaseChartLayer) renderCandlesticks(chart *ChartWidget) {
	rect := chart.GetWidget().Rect
	minX, minY := float32(0), float32(0)
	maxX, maxY := float32(rect.Dx()), float32(rect.Dy())
	candleWidth := float32(chart.barWidth * 0.8)

	visibleBars := float64(rect.Dx()) / chart.barWidth
	visibleStartBar := chart.barOffset
	visibleEndBar := visibleStartBar + visibleBars
	startIndex := int(max(0, visibleStartBar))
	endIndex := min(len(l.candles)-2, int(visibleEndBar))

	// If no visible data, skip
	if startIndex > endIndex {
		return
	}

	var vertices []ebiten.Vertex
	var indices []uint16
	idxCount := uint16(0)

	addRect := func(x, y, w, h float32, col color.Color) {
		if w <= 0 || h <= 0 {
			return
		}

		r, g, b, a := col.RGBA()
		colR := float32(r>>8) / 255
		colG := float32(g>>8) / 255
		colB := float32(b>>8) / 255
		colA := float32(a>>8) / 255

		// top-left
		vertices = append(vertices, ebiten.Vertex{
			DstX: x, DstY: y,
			SrcX: 0, SrcY: 0,
			ColorR: colR, ColorG: colG, ColorB: colB, ColorA: colA,
		})
		// top-right
		vertices = append(vertices, ebiten.Vertex{
			DstX: x + w, DstY: y,
			SrcX: 1, SrcY: 0,
			ColorR: colR, ColorG: colG, ColorB: colB, ColorA: colA,
		})
		// bottom-left
		vertices = append(vertices, ebiten.Vertex{
			DstX: x, DstY: y + h,
			SrcX: 0, SrcY: 1,
			ColorR: colR, ColorG: colG, ColorB: colB, ColorA: colA,
		})
		// bottom-right
		vertices = append(vertices, ebiten.Vertex{
			DstX: x + w, DstY: y + h,
			SrcX: 1, SrcY: 1,
			ColorR: colR, ColorG: colG, ColorB: colB, ColorA: colA,
		})

		indices = append(indices,
			idxCount, idxCount+1, idxCount+2,
			idxCount+1, idxCount+3, idxCount+2,
		)
		idxCount += 4
	}

	clip := func(val float32) float32 {
		if val < minY {
			val = minY
		} else if val > maxY {
			val = maxY
		}
		return val
	}

	for i := startIndex; i <= endIndex; i++ {
		c := l.candles[i]
		// Convert candle’s time → bar index
		barIndex := float64(chart.getBarIndex(c.Unix))

		// The raw left/right in pixels
		left := minX + float32((barIndex-chart.barOffset)*chart.barWidth)
		right := left + candleWidth

		// Partially clamp horizontally
		clippedLeft := float32(math.Max(float64(left), float64(minX)))
		clippedRight := float32(math.Min(float64(right), float64(maxX)))
		width := clippedRight - clippedLeft
		if width <= 0 {
			// Entirely offscreen horizontally
			continue
		}

		// Compute candle Y’s
		openY := chart.getPriceY(c.Open)
		closeY := chart.getPriceY(c.Close)
		highY := chart.getPriceY(c.High)
		lowY := chart.getPriceY(c.Low)

		// Candle color
		col := settings.CandleStickGreen
		if c.Close < c.Open {
			col = settings.CandleStickRed
		}

		// Top/bottom numeric values (remember Ebiten’s Y grows downward)
		topY := float32(math.Min(float64(openY), float64(closeY)))
		botY := float32(math.Max(float64(openY), float64(closeY)))

		// Clamp wicks & body vertically
		highY = clip(highY) // clip() ensures within [minY, maxY]
		lowY = clip(lowY)
		topY = clip(topY)
		botY = clip(botY)

		bodyHeight := botY - topY
		if bodyHeight < 1 {
			bodyHeight = 1
		}

		// Draw body using clamped horizontal coords
		addRect(clippedLeft, topY, width, bodyHeight, col)

		// The center for the wick is the horizontal midpoint of the visible body
		midX := left + candleWidth*0.5
		wickX := midX - 0.5

		if wickX < maxX {
			// Top wick
			if topY > highY {
				addRect(wickX, highY, 1, topY-highY, col)
			}
			// Bottom wick
			if botY < lowY {
				addRect(wickX, botY, 1, lowY-botY, col)
			}
		}
	}

	if len(vertices) > 0 && len(indices) > 0 {
		op := &ebiten.DrawTrianglesOptions{}
		op.Blend = ebiten.BlendSourceOver
		l.image.DrawTriangles(vertices, indices, l.triImage, op)
	}
}

func (l *BaseChartLayer) renderLines(chart *ChartWidget) {
	rect := chart.GetWidget().Rect
	minY := float32(0)
	maxY := float32(rect.Max.Y)
	minX := float32(0)
	maxX := float32(rect.Max.X)

	for i := 0; i < len(l.candles)-2; i++ {
		t1 := l.candles[i]
		t2 := l.candles[i+1]

		barIndex1 := float64(chart.getBarIndex(t1.Unix))
		barIndex2 := float64(chart.getBarIndex(t2.Unix))

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

		vector.StrokeLine(l.image,
			x1,
			y1,
			x2,
			y2,
			float32(settings.LineChartStrokeWidth), settings.LineChartLineColor, true)
	}
}

func (l *BaseChartLayer) render(screen *ebiten.Image, chart *ChartWidget) {
	if len(l.candles) == 0 || !l.visible || l.image == nil {
		return
	}
	if chart.isDirty || l.isDirty {
		l.image.Clear()
		l.renderUpdate(chart)
	}
	if chart.chartType == chartTypeCandles {
		l.renderLastCandle(screen, chart)
	}
	if chart.chartType == chartTypeLine {
		l.renderLastLine(screen, chart)
	}

	op := &ebiten.DrawImageOptions{}
	op.Blend = ebiten.BlendSourceOver
	op.GeoM.Translate(float64(chart.GetWidget().Rect.Min.X), float64(chart.GetWidget().Rect.Min.Y))

	screen.DrawImage(l.image, op)
}

// TODO: 1 draw call
func (l *BaseChartLayer) renderVolumes(chart *ChartWidget) {
	rect := chart.GetWidget().Rect
	minX, _ := float32(0), float32(0)
	maxY := float32(rect.Dy())
	candleWidth := float32(chart.barWidth * 0.8)

	visibleBars := float64(rect.Dx()) / chart.barWidth
	visibleStartBar := chart.barOffset
	visibleEndBar := visibleStartBar + visibleBars
	startIndex := int(max(0, visibleStartBar))
	endIndex := min(len(l.candles)-2, int(visibleEndBar))

	// If no visible data, skip
	if startIndex > endIndex {
		return
	}

	var maxVol float64 = 0
	for i := startIndex; i <= endIndex; i++ {
		c := l.candles[i]
		vol := math.Log1p(c.Vbuy + c.Vsell)
		if vol > maxVol {
			maxVol = vol
		}
	}

	for i := startIndex; i <= endIndex; i++ {
		c := l.candles[i]
		barIndex := float64(chart.getBarIndex(c.Unix))
		left := minX + float32((barIndex-chart.barOffset)*chart.barWidth)

		barHeight := maxY * settings.VolumeBarHeightPerc
		normalizedHeightBuy := float32((math.Log1p(c.Vbuy) / maxVol) * float64(barHeight))
		normalizedHeightSell := float32((math.Log1p(c.Vsell) / maxVol) * float64(barHeight))

		if normalizedHeightBuy > barHeight {
			normalizedHeightBuy = barHeight
		}
		if normalizedHeightSell > barHeight {
			normalizedHeightSell = barHeight
		}

		vector.DrawFilledRect(l.image, left, maxY-normalizedHeightBuy, candleWidth/2, normalizedHeightBuy, settings.VolumeBarGreen, false)
		vector.DrawFilledRect(l.image, left+candleWidth/2, maxY-normalizedHeightSell, candleWidth/2, normalizedHeightSell, settings.VolumeBarRed, false)
	}
}

func (l *BaseChartLayer) receiveData() {
	for ev := range l.eventCh {
		switch msg := ev.(type) {
		case event.Candle:
			if l.chart.lastUnix+l.chart.interval <= msg.Unix || len(l.candles) == 0 {
				l.candles = append(l.candles, msg)
				l.chart.lastUnix = msg.Unix
				l.isDirty = true
			}
			l.candles[len(l.candles)-1] = msg
			l.chart.lastPrice = msg.Close
		}
	}
	fmt.Println("stopped receive data loop")
}

func (l *BaseChartLayer) renderLastLine(screen *ebiten.Image, chart *ChartWidget) {
	rect := chart.screen.GetWidget().Rect
	minX := float64(rect.Min.X)
	maxX := float64(rect.Max.X)
	minY := float64(rect.Min.Y)
	maxY := float64(rect.Max.Y)

	if len(l.candles) > 1 {
		c1 := l.candles[len(l.candles)-2]
		c2 := l.candles[len(l.candles)-1]
		prevBarIndex := float64(chart.getBarIndex(c1.Unix))
		currentBarIndex := float64(chart.getBarIndex(c2.Unix))

		visibleBars := float64(rect.Dx()) / chart.barWidth
		visibleStartBar := chart.barOffset
		visibleEndBar := visibleStartBar + visibleBars

		if currentBarIndex >= visibleStartBar && currentBarIndex <= visibleEndBar {
			x1 := math.Max(math.Min(minX+(prevBarIndex-chart.barOffset)*chart.barWidth, maxX), minX)
			x2 := math.Max(math.Min(minX+(currentBarIndex-chart.barOffset)*chart.barWidth, maxX), minX)
			y1 := math.Max(math.Min(float64(chart.getPriceYScreen(c1.Close)), maxY), minY)
			y2 := math.Max(math.Min(float64(chart.getPriceYScreen(c2.Close)), maxY), minY)

			vector.StrokeLine(screen,
				float32(x1),
				float32(y1),
				float32(x2),
				float32(y2),
				float32(settings.LineChartStrokeWidth), settings.LineChartLineColor, true)
		}
	}
}

func (l *BaseChartLayer) onChartTypeChange(_ any) {
	l.isDirty = true
}

func (l *BaseChartLayer) onIntervalChange(interval any) {
	_, ok := interval.(int64)
	if !ok {
		log.Fatal("interval changed failed cast", interval)
		return
	}
	app.engine.Poison(l.sessionPID)

	l.eventCh = make(chan any)
	streams := []session.Stream{{
		Stream:    event.StreamCandles,
		Timeframe: l.chart.interval,
	}}

	l.sessionPID = app.engine.Spawn(session.New(l.eventCh, l.chart.pair, streams), "session")
	l.candles = []event.Candle{}
	l.isDirty = true

	go l.receiveData()
}

// TODO: this might be optimized even further, but its good for now.
func (l *BaseChartLayer) renderLastCandle(screen *ebiten.Image, chart *ChartWidget) {
	rect := chart.screen.GetWidget().Rect
	minX, minY := float32(rect.Min.X), float32(rect.Min.Y)
	maxX, maxY := float32(rect.Max.X), float32(rect.Max.Y)
	candleWidth := float32(chart.barWidth * 0.8)

	i := len(l.candles) - 1
	c := l.candles[i]
	barIndex := chart.getBarIndex(c.Unix)
	x := minX + float32((float64(barIndex)-chart.barOffset)*chart.barWidth)

	openY := chart.getPriceYScreen(c.Open)
	closeY := chart.getPriceYScreen(c.Close)
	highY := chart.getPriceYScreen(c.High)
	lowY := chart.getPriceYScreen(c.Low)

	// Determine candle color
	col := settings.CandleStickGreen
	if c.Close < c.Open {
		col = settings.CandleStickRed
	}

	// Body top/bottom
	topY := float32(math.Min(float64(openY), float64(closeY)))
	botY := float32(math.Max(float64(openY), float64(closeY)))

	// Skip drawing if completely off-screen horizontally
	if x+candleWidth < minX || x > maxX {
		return
	}

	// Clamp x coordinates
	if x < minX {
		x = minX
	}
	rightX := x + candleWidth
	if rightX > maxX {
		rightX = maxX
	}

	// Clamp y coordinates
	clip := func(val float32) float32 {
		return float32(math.Max(float64(minY), math.Min(float64(val), float64(maxY))))
	}
	highY, lowY = clip(highY), clip(lowY)
	topY, botY = clip(topY), clip(botY)

	bodyHeight := botY - topY
	vector.DrawFilledRect(screen, x, topY, rightX-x, bodyHeight, col, false)

	midX := float32(math.Min(float64(x+candleWidth*0.5), float64(maxX)))
	vector.StrokeLine(screen, midX, highY, midX, topY, 1, col, false)
	vector.StrokeLine(screen, midX, botY, midX, lowY, 1, col, false)
}
