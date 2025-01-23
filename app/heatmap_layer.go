package app

import (
	"fmt"
	"image/color"
	"log"
	"marketmonkey/actor/session"
	"marketmonkey/event"
	"marketmonkey/settings"
	"math"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/colornames"
)

type FlattenedHeatmapLevel struct {
	Unix      int64
	Price     float64
	Size      float64
	Intensity float64
}

type HeatmapLayer struct {
	pair       event.Pair
	name       string
	eventCh    chan any
	streams    []session.Stream
	sessionPID *actor.PID

	lastUnix int64

	image    *ebiten.Image
	vertImg  *ebiten.Image
	isDirty  bool
	heats    []event.Heatmap
	interval int64
}

func NewHeatmapLayer(pair event.Pair) *HeatmapLayer {
	eventCh := make(chan any)
	streams := []session.Stream{{
		Stream: event.StreamHeatmap,
	}}
	pid := app.engine.Spawn(session.New(eventCh, pair, streams), "session")

	vertImg := ebiten.NewImage(1, 1)
	vertImg.Fill(colornames.White)

	layer := &HeatmapLayer{
		name:       fmt.Sprintf("Heatmap - %s %s", pair.Exchange, pair.Symbol),
		pair:       pair,
		streams:    streams,
		sessionPID: pid,
		eventCh:    eventCh,
		vertImg:    vertImg,
		lastUnix:   time.Now().Unix(),
		heats:      []event.Heatmap{},
	}

	go layer.receiveData()

	return layer
}

func (l *HeatmapLayer) initialize(chart *ChartWidget) {
	l.interval = chart.interval
	chart.intervalChangeEvent.AddHandler(l.onIntervalChange)
}

func (l *HeatmapLayer) receiveData() {
	fmt.Println("heatmap start recieving data")
	for ev := range l.eventCh {
		switch msg := ev.(type) {
		case event.Heatmap:
			if l.lastUnix+l.interval <= msg.Unix {
				l.heats = append(l.heats, msg)
				l.isDirty = true
				l.lastUnix = msg.Unix
			}
		}
	}
	fmt.Println("heatmap stopped recieving data")
}

func (l *HeatmapLayer) update(chart *ChartWidget) {
	if l.image == nil {
		l.image = ebiten.NewImage(chart.GetWidget().Rect.Dx(), chart.GetWidget().Rect.Dy())
		l.image.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0})
	}
}

func (l *HeatmapLayer) renderUpdate(chart *ChartWidget) {
	l.isDirty = false
	rect := chart.GetWidget().Rect

	visibleBars := float64(rect.Dx()) / chart.barWidth
	visibleStartBar := chart.barOffset
	visibleEndBar := visibleStartBar + visibleBars
	startIndex := int(max(0, visibleStartBar))
	endIndex := min(len(l.heats)-1, int(visibleEndBar))

	if startIndex > endIndex {
		return
	}

	// Use int for idxCount to avoid uint16 wraparound
	var idxCount int

	// We'll store vertices/indices as usual, but remember
	// we convert idxCount -> uint16 only at append-time.
	var (
		vertices  []ebiten.Vertex
		indices   []uint16
		textDraws []struct {
			x, y    float32
			sizeStr string
		}
	)

	// 65535 is the max for a 16-bit index, so we keep some margin
	const maxIndexCount = 65532

	drawBatch := func() {
		if len(vertices) > 0 && len(indices) > 0 {
			//			fmt.Printf(":: BATCHING :: verts %d indices %d\n", len(vertices), len(indices))
			op := &ebiten.DrawTrianglesOptions{}
			op.Blend = ebiten.BlendSourceOver
			l.image.DrawTriangles(vertices, indices, l.vertImg, op)
		}
		vertices = vertices[:0]
		indices = indices[:0]
		idxCount = 0
	}

	for i := startIndex; i <= endIndex; i++ {
		heat := l.heats[i]
		barIndex := float64(chart.getBarIndex(heat.Unix))

		for _, level := range heat.Levels {
			botPrice := level.Price
			topPrice := level.Price + float64(heat.PriceGroup)

			yTop := chart.getPriceY(topPrice)
			yBot := chart.getPriceY(botPrice)

			rectY := float32(math.Min(float64(yTop), float64(yBot)))
			rectH := float32(math.Abs(float64(yBot - yTop)))

			rectX := float32((barIndex - chart.barOffset) * chart.barWidth)
			rectW := float32(chart.barWidth * 0.95)

			// Skip if out of Y-bounds
			if rectY+rectH < float32(0) || rectY > float32(rect.Max.Y) {
				continue
			}
			// Clamp Y
			if rectY < float32(0) {
				diff := float32(0) - rectY
				rectY = float32(0)
				rectH -= diff
			}
			if rectY+rectH > float32(rect.Max.Y) {
				rectH = float32(rect.Max.Y) - rectY
			}

			// Clamp X
			if rectX < float32(0) {
				diff := float32(0) - rectX
				rectX = float32(0)
				rectW -= diff
			}
			if rectX+rectW > float32(rect.Max.X) {
				rectW = float32(rect.Max.X) - rectX
			}

			if rectW <= 0 || rectH <= 0 {
				continue
			}

			if idxCount+4 > maxIndexCount {
				drawBatch()
			}

			col := GetIntensityColorNew(float32(level.Intensity))

			tl := ebiten.Vertex{
				DstX:   rectX,
				DstY:   rectY,
				SrcX:   0,
				SrcY:   0,
				ColorR: float32(col.R) / 255,
				ColorG: float32(col.G) / 255,
				ColorB: float32(col.B) / 255,
				ColorA: float32(col.A) / 255,
			}
			tr := ebiten.Vertex{
				DstX:   rectX + rectW,
				DstY:   rectY,
				SrcX:   1,
				SrcY:   0,
				ColorR: tl.ColorR, ColorG: tl.ColorG, ColorB: tl.ColorB, ColorA: tl.ColorA,
			}
			bl := ebiten.Vertex{
				DstX:   rectX,
				DstY:   rectY + rectH,
				SrcX:   0,
				SrcY:   1,
				ColorR: tl.ColorR, ColorG: tl.ColorG, ColorB: tl.ColorB, ColorA: tl.ColorA,
			}
			br := ebiten.Vertex{
				DstX:   rectX + rectW,
				DstY:   rectY + rectH,
				SrcX:   1,
				SrcY:   1,
				ColorR: tl.ColorR, ColorG: tl.ColorG, ColorB: tl.ColorB, ColorA: tl.ColorA,
			}

			// Append the 4 vertices
			vertices = append(vertices, tl, tr, bl, br)

			// Convert idxCount to uint16 only when appending indices
			base := uint16(idxCount)
			indices = append(indices,
				base+0, base+1, base+2,
				base+2, base+1, base+3,
			)
			idxCount += 4

			if rectH-4 > float32(settings.HeatmapSizeTextHeight) && rectW-4 > float32(settings.HeatmapSizeTextWidth) {
				sizeLabel := fmt.Sprintf("%.2f", level.Size)
				textDraws = append(textDraws, struct {
					x, y    float32
					sizeStr string
				}{
					x:       rectX + rectW/2 - float32(settings.HeatmapSizeTextWidth)/2,
					y:       rectY + rectH/2 - float32(settings.HeatmapSizeTextHeight)/2,
					sizeStr: sizeLabel,
				})
			}

		}
	}

	drawBatch()

	for _, drawInfo := range textDraws {
		ops := text.DrawOptions{}
		ops.GeoM.Translate(float64(drawInfo.x), float64(drawInfo.y))
		text.Draw(l.image, drawInfo.sizeStr, settings.FontSM, &ops)
	}
}

func (l *HeatmapLayer) render(screen *ebiten.Image, chart *ChartWidget) {
	if l.image == nil {
		return
	}
	if chart.isDirty || l.isDirty {
		l.image.Clear()
		l.renderUpdate(chart)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(chart.GetWidget().Rect.Min.X), float64(chart.GetWidget().Rect.Min.Y))
	op.Blend = ebiten.BlendSourceOver
	screen.DrawImage(l.image, op)
}

func (l *HeatmapLayer) delete() {
	app.engine.Poison(l.sessionPID)
}

func (l *HeatmapLayer) onIntervalChange(interval any) {
	i, ok := interval.(int64)
	if !ok {
		log.Fatal("interval changed failed cast", interval)
		return
	}

	l.interval = i
	l.heats = []event.Heatmap{}
	l.isDirty = true
}
