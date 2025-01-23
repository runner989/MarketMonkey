package app

import (
	"marketmonkey/settings"
	"math"
	"time"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/colornames"
)

type TimeScaleWidget struct {
	*widget.Container

	chart *ChartWidget
}

func NewTimeScale(chart *ChartWidget) *TimeScaleWidget {
	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, int(settings.ChartTimeScaleHeight)),
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionStart,
			}),
		),
	)
	return &TimeScaleWidget{
		Container: container,
		chart:     chart,
	}
}

func (ts *TimeScaleWidget) Render(screen *ebiten.Image) {
	// First, render the container's background
	ts.Container.Render(screen)

	rect := ts.GetWidget().Rect
	minX := float64(rect.Min.X)
	maxX := float64(rect.Max.X)

	// Convert screen X → bar indices
	minBar := ts.chart.barOffset + (minX-float64(rect.Min.X))/ts.chart.barWidth
	maxBar := ts.chart.barOffset + (maxX-float64(rect.Min.X))/ts.chart.barWidth

	// Bar indices → time in seconds
	minT := minBar * float64(ts.chart.interval)
	maxT := maxBar * float64(ts.chart.interval)
	if minT > maxT {
		minT, maxT = maxT, minT
	}

	// Compute step, start/end
	step := pickTimeInterval(maxT - minT)
	stepSec := step.Seconds()
	start := math.Floor(minT/stepSec) * stepSec
	end := math.Ceil(maxT/stepSec) * stepSec

	for secs := start; secs <= end; secs += stepSec {
		barIndex := secs / float64(ts.chart.interval)
		x := float64(rect.Min.X) + (barIndex-ts.chart.barOffset)*ts.chart.barWidth

		if x < minX || x > maxX {
			continue
		}

		// Debug exact time lines
		// chartScreenRect := ts.chart.GetWidget().Rect
		// vector.StrokeLine(screen, float32(x), float32(rect.Min.Y),
		// 	float32(x), float32(chartScreenRect.Min.Y),
		// 	1, colornames.Gray, true)

		label := ts.chart.startTime.Add(time.Duration(secs) * time.Second).Format("15:04:05")
		font := settings.FontSM
		drawX := x
		drawY := float64(rect.Min.Y) + float64(font.Metrics().CapHeight)

		DrawText(screen, label, font, drawX, drawY, colornames.White)
	}
}
