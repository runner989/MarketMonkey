package app

import (
	"fmt"
	img "image"
	"image/color"
	"math"
	"time"

	evt "marketmonkey/event"
	"marketmonkey/settings"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/exp/shiny/materialdesign/colornames"
)

const (
	minTimeZoom  = 0.01
	maxTimeZoom  = 100.0
	minPriceZoom = 0.01
	maxPriceZoom = 100.0
)

type chartType int

const (
	chartTypeCandles chartType = iota
	chartTypeLine
)

type ChartWidget struct {
	*widget.Container

	pair        evt.Pair
	timeZoom    float64
	priceZoom   float64
	priceRange  float64
	centerPrice float64

	basePixelsPerUnix float32
	pixelsPerUnix     float32

	visiblePriceRange float64
	minPrice          float64
	maxPrice          float64
	lastPrice         float64
	lastUnix          int64

	screen     *widget.Container
	priceScale *PriceScaleWidget
	timeScale  *TimeScaleWidget
	startTime  time.Time

	layers []layer

	isPanning       bool
	isMouseInBounds bool
	isDirty         bool

	baseBarWidth float64
	barWidth     float64
	barOffset    float64
	interval     int64

	// last dragged pos
	lastPos img.Point

	intervalChangeEvent  *event.Event
	chartTypeChangeEvent *event.Event

	// the chart type of the base layer (candles, line, heiken)
	chartType chartType
	baseLayer *BaseChartLayer
}

func NewChartWidget(pair evt.Pair, interval int64) *ChartWidget {
	chart := ChartWidget{
		pixelsPerUnix:        30,
		basePixelsPerUnix:    30,
		timeZoom:             1,
		priceZoom:            0.5,
		layers:               []layer{},
		baseBarWidth:         30,
		barWidth:             30,
		interval:             interval,
		priceRange:           float64(settings.ChartPriceScaleDefaultPriceRange),
		chartType:            chartTypeCandles,
		barOffset:            -10,
		intervalChangeEvent:  &event.Event{},
		chartTypeChangeEvent: &event.Event{},
		pair:                 pair,
	}
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.BackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Stretch([]bool{true, false}, []bool{true, true}),
			widget.GridLayoutOpts.Spacing(1, 0), // Optional: adds spacing between columns
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchHorizontal:  true,
				StretchVertical:    true,
			}),
		),
	)
	leftContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.BackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true, false}),
			widget.GridLayoutOpts.Spacing(0, 1),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, 0),
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionStart,
				VerticalPosition:   widget.GridLayoutPositionStart,
			}),
		),
	)
	screenContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.CursorMoveHandler(chart.onMouseMove),
			widget.WidgetOpts.MouseButtonPressedHandler(chart.onMousePressed),
			widget.WidgetOpts.MouseButtonReleasedHandler(chart.onMouseReleased),
			widget.WidgetOpts.ScrolledHandler(chart.onScroll),
			widget.WidgetOpts.CursorEnterHandler(chart.onContainerEnter),
			widget.WidgetOpts.CursorExitHandler(chart.onContainerLeave),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				StretchHorizontal: true,
				StretchVertical:   true,
			}),
		),
	)
	chart.screen = screenContainer
	chart.timeScale = NewTimeScale(&chart)
	leftContainer.AddChild(screenContainer, chart.timeScale)
	chart.priceScale = NewPriceScale(&chart)
	rootContainer.AddChild(leftContainer, chart.priceScale)
	chart.Container = rootContainer

	chart.baseLayer = NewBaseChartLayer(&chart)

	return &chart
}

func (chart *ChartWidget) AddLayer(layers ...layer) {
	for _, layer := range layers {
		chart.layers = append(chart.layers, layer)
		layer.initialize(chart)
	}
}

func (chart *ChartWidget) GetWidget() *widget.Widget {
	return chart.screen.GetWidget()
}

// TODO: optimize this to only calculate when needed
func (chart *ChartWidget) Update() {
	chart.Container.Update()

	// HACK: around updating the chart that its dragged so
	// layers can rerender if needed.
	// TODO: maybe find another way ? or not.
	point := chart.GetWidget().Rect.Min
	if point.X != chart.lastPos.X || point.Y != chart.lastPos.Y {
		chart.isDirty = true
		chart.lastPos = point
	}

	chart.visiblePriceRange = chart.priceRange / chart.priceZoom
	chart.minPrice = chart.centerPrice - (chart.visiblePriceRange / 2)
	chart.maxPrice = chart.centerPrice + (chart.visiblePriceRange / 2)

	if chart.centerPrice == 0 {
		chart.centerPrice = chart.lastPrice
	}

	chart.baseLayer.update(chart)
	for _, layer := range chart.layers {
		layer.update(chart)
	}
}

func (chart *ChartWidget) onMouseMove(args *widget.WidgetCursorMoveEventArgs) {
	if chart.isPanning {
		deltaX := args.DiffX
		deltaY := args.DiffY

		// price
		chartHeight := chart.GetWidget().Rect.Dy()
		visiblePriceRange := chart.priceRange / chart.priceZoom
		pricePerPixel := visiblePriceRange / float64(chartHeight)
		chart.centerPrice += float64(deltaY) * pricePerPixel

		// Horizontal pan in bar space
		barDelta := float64(deltaX) / float64(chart.barWidth)
		chart.barOffset -= barDelta

		chart.isDirty = true
	}
}

func (chart *ChartWidget) onMousePressed(args *widget.WidgetMouseButtonPressedEventArgs) {
	if args.Button == settings.PanChartButton {
		chart.isPanning = true
	}
}

func (chart *ChartWidget) onMouseReleased(args *widget.WidgetMouseButtonReleasedEventArgs) {
	if args.Button == settings.PanChartButton {
		chart.isPanning = false
	}
}

func (chart *ChartWidget) onContainerEnter(_ *widget.WidgetCursorEnterEventArgs) {
	chart.isMouseInBounds = true
}

func (chart *ChartWidget) onContainerLeave(_ *widget.WidgetCursorExitEventArgs) {
	chart.isMouseInBounds = false
}

func (chart *ChartWidget) onScroll(args *widget.WidgetScrolledEventArgs) {
	scrollY := args.Y

	if !ebiten.IsKeyPressed(ebiten.KeyA) {
		old := chart.priceZoom
		factor := 1.05
		if scrollY > 0 {
			chart.priceZoom *= factor
		} else if scrollY < 0 {
			chart.priceZoom /= factor
		}
		// Clamp so we don’t go insane
		if chart.priceZoom < minPriceZoom {
			chart.priceZoom = minPriceZoom
		} else if chart.priceZoom > maxPriceZoom {
			chart.priceZoom = maxPriceZoom
		}
		// Re-center around mouse Y
		_, mouseY := ebiten.CursorPosition()
		priceAtMouse := chart.getPriceAtY(float32(mouseY))
		ratio := old / chart.priceZoom
		chart.centerPrice = priceAtMouse + (chart.centerPrice-priceAtMouse)*ratio
	} else {
		oldBarWidth := chart.barWidth
		factor := 1.05
		if scrollY > 0 {
			chart.barWidth *= factor
		} else if scrollY < 0 {
			chart.barWidth /= factor
		}

		if chart.barWidth < 10 {
			chart.barWidth = 10
		} else if chart.barWidth > 100 {
			chart.barWidth = 100
		}

		// Re-center around mouseX if desired
		mouseX, _ := ebiten.CursorPosition()
		rect := chart.GetWidget().Rect
		oldBarIndex := (float64(mouseX)-float64(rect.Min.X))/oldBarWidth + chart.barOffset
		zoomRatio := oldBarWidth / chart.barWidth
		chart.barOffset = oldBarIndex - (oldBarIndex-chart.barOffset)*zoomRatio
	}

	chart.isDirty = true
}

func (chart *ChartWidget) Render(screen *ebiten.Image) {
	chart.Container.Render(screen)

	chart.renderCrosshair(screen)
	chart.renderPriceLine(screen)

	for _, layer := range chart.layers {
		layer.render(screen, chart)
	}
	chart.baseLayer.render(screen, chart)
	chart.isDirty = false
}

func (chart *ChartWidget) renderPriceLine(screen *ebiten.Image) {
	if !settings.ChartRenderPriceLine {
		return
	}
	rect := chart.GetWidget().Rect
	x1 := float32(rect.Min.X)
	x2 := float32(rect.Max.X)
	y := chart.getPriceYScreen(chart.lastPrice)

	DrawDashedLine(screen, x1, y, x2, y, 0.5, 5, 5, colornames.Blue100, true)

	yLabel := chart.getPriceYScreen(chart.lastPrice)
	vector.DrawFilledRect(
		screen,
		float32(rect.Max.X),
		y-float32(settings.ChartPriceLabelHeight)/2,
		float32(settings.ChartPriceLabelWidth),
		float32(settings.ChartPriceLabelHeight),
		settings.ChartPriceLabelColor,
		false,
	)

	label := fmt.Sprintf("%.2f", chart.lastPrice)
	x := float64(chart.GetWidget().Rect.Max.X) + float64(settings.ChartPriceScaleMargin)
	font := settings.FontSM
	_, fh := text.Measure(label, font, font.Metrics().VLineGap)
	DrawText(screen, label, settings.FontSM, float64(x), float64(yLabel)-fh/2, color.Black)
}

func (chart *ChartWidget) renderCrosshair(screen *ebiten.Image) {
	if !chart.isMouseInBounds {
		return
	}
	mx, my := ebiten.CursorPosition()
	rect := chart.GetWidget().Rect
	vector.StrokeLine(screen,
		float32(rect.Min.X),
		float32(my),
		float32(rect.Max.X),
		float32(my),
		1, settings.ChartCrossHairColor, false)
	vector.StrokeLine(screen,
		float32(mx),
		float32(rect.Min.Y),
		float32(mx),
		float32(rect.Max.Y),
		1, settings.ChartCrossHairColor, false)
	vector.DrawFilledRect(
		screen,
		float32(rect.Max.X),
		float32(my)-settings.ChartPriceLabelHeight/2,
		float32(settings.ChartPriceLabelWidth),
		float32(settings.ChartPriceLabelHeight),
		colornames.White, false)
	price := chart.getPriceAtY(float32(my))
	label := fmt.Sprintf("%.2f", price)
	font := settings.FontSM
	_, fh := text.Measure(label, font, font.Metrics().VLineGap)
	x := float64(rect.Max.X) + float64(settings.PanelPadding)
	y := float64(my) - fh/2
	DrawText(screen, label, font, x, y, colornames.Black)

	t := chart.getTimeAtX(float64(mx))
	tLabel := chart.startTime.Add(time.Duration(t) * time.Second).Format("15:04:05")
	vector.DrawFilledRect(
		screen,
		float32(mx)-settings.ChartPriceLabelWidth/2,
		float32(rect.Max.Y),
		float32(settings.ChartPriceLabelWidth),
		float32(settings.ChartPriceLabelHeight),
		colornames.White, false)

	fw, fh := text.Measure(tLabel, font, font.Metrics().VLineGap)
	x = float64(float64(mx) - fw/2)
	y = float64(rect.Max.Y) + fh/4
	DrawText(screen, tLabel, font, x, y, colornames.Black)
}

func (chart *ChartWidget) getPriceYScreen(price float64) float32 {
	rect := chart.GetWidget().Rect
	priceRange := chart.maxPrice - chart.minPrice
	if priceRange <= 0 {
		return float32(rect.Min.Y + rect.Dy()/2)
	}
	pixelsPerPrice := float32(rect.Dy()) / float32(priceRange)
	// We want minPrice at bottom, maxPrice at top → invert
	offset := float32(price-chart.minPrice) * pixelsPerPrice
	return float32(rect.Min.Y+rect.Dy()) - offset
}

func (chart *ChartWidget) getPriceY(price float64) float32 {
	h := float32(chart.GetWidget().Rect.Dy())
	priceRange := chart.maxPrice - chart.minPrice
	if priceRange <= 0 {
		return h / 2
	}
	pixelsPerPrice := h / float32(priceRange)
	offset := float32(price-chart.minPrice) * pixelsPerPrice
	return h - offset
}

func (chart *ChartWidget) getPriceAtY(y float32) float64 {
	height := float32(chart.GetWidget().Rect.Dy())
	relY := float64(height - y)
	pricePerPixel := chart.visiblePriceRange / float64(height)
	return chart.minPrice + relY*pricePerPixel
}

func (chart *ChartWidget) getBarIndex(unix int64) int64 {
	return int64(float64(unix-chart.startTime.Unix()) / float64(chart.interval))
}

func (chart *ChartWidget) getTimeAtX(x float64) float64 {
	rect := chart.GetWidget().Rect
	barIndex := (x-float64(rect.Min.X))/chart.barWidth + chart.barOffset
	return float64(barIndex) * float64(chart.interval)
}

func (chart *ChartWidget) Close(_ *widget.WindowClosedEventArgs) {
	for _, layer := range chart.layers {
		layer.delete()
	}
}

func (chart *ChartWidget) onIntervalChange(interval int64) {
	if chart.interval != interval {
		chart.interval = interval
		chart.intervalChangeEvent.Fire(interval)
	}
}

func (chart *ChartWidget) Toolbar() *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	comboBox := intervalDropdown(chart.onIntervalChange)
	container.AddChild(comboBox)

	buttonLine := newToolbarButton("L")
	buttonCandle := newToolbarButton("C")
	buttonCandle.TextColor.Idle = settings.MenuButtonTextColorActive
	buttonLine.ClickedEvent.AddHandler(func(_ any) {
		if chart.chartType != chartTypeLine {
			buttonLine.TextColor.Idle = settings.MenuButtonTextColorActive
			buttonCandle.TextColor.Idle = settings.MenuButtonTextColorIdle
			chart.chartType = chartTypeLine
			chart.chartTypeChangeEvent.Fire(chartTypeLine)
		} else {
			buttonLine.TextColor.Idle = settings.MenuButtonTextColorIdle
		}
	})
	buttonCandle.ClickedEvent.AddHandler(func(_ any) {
		if chart.chartType != chartTypeCandles {
			buttonCandle.TextColor.Idle = settings.MenuButtonTextColorActive
			buttonLine.TextColor.Idle = settings.MenuButtonTextColorIdle
			chart.chartType = chartTypeCandles
			chart.chartTypeChangeEvent.Fire(chartTypeCandles)
		} else {
			buttonCandle.TextColor.Idle = settings.MenuButtonTextColorIdle
		}
	})
	container.AddChild(buttonCandle, buttonLine)
	return container
}

func pickTimeInterval(secondsVisible float64) time.Duration {
	steps := []time.Duration{
		time.Second,
		5 * time.Second,
		15 * time.Second,
		30 * time.Second,
		time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		time.Hour,
		4 * time.Hour,
		24 * time.Hour,
	}
	for _, s := range steps {
		if secondsVisible/s.Seconds() < 10 {
			return s
		}
	}
	return 24 * time.Hour
}

func getDynamicPriceStep(visibleRange float64) float64 {
	magnitude := math.Floor(math.Log10(visibleRange))
	baseStep := math.Pow(10, magnitude)
	normalizedRange := visibleRange / baseStep
	var step float64
	switch {
	case normalizedRange <= 1.0:
		step = baseStep / 20
	case normalizedRange <= 2.0:
		step = baseStep / 10
	case normalizedRange <= 5.0:
		step = baseStep / 4
	case normalizedRange <= 10.0:
		step = baseStep / 2
	default:
		step = baseStep
	}
	// "Nice" steps
	if step >= 1 {
		possibleSteps := []float64{1, 2, 5, 10, 20, 25, 50, 100, 200, 500, 1000}
		for _, ps := range possibleSteps {
			if visibleRange/ps >= 4 && visibleRange/ps <= 15 {
				step = ps
				break
			}
		}
	}
	return step
}
