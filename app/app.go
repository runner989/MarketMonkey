package app

import (
	img "image"
	"marketmonkey/event"
	"marketmonkey/settings"
	"math"

	"github.com/anthdm/hollywood/actor"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

var app *App

type App struct {
	ui *ebitenui.UI

	contentContainer *widget.Container
	engine           *actor.Engine
}

func New(e *actor.Engine) *App {
	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.BackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Spacing(0, 0),
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch(
				[]bool{true, true, true},
				[]bool{false, true, false}),
		)),
	)
	content := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.BackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	app = &App{
		ui: &ebitenui.UI{
			Container: root,
		},
		contentContainer: content,
		engine:           e,
	}

	root.AddChild(NewMenuBarWidget(), content, NewStatusBarWidget())

	return app
}

func (app *App) Draw(screen *ebiten.Image) {
	app.ui.Draw(screen)
}

var (
	timer   float64
	elapsed bool
)

func (app *App) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	app.ui.Update()

	if !elapsed {
		timer += 1.0 / 60.0
		if timer > 1 {
			app.loadInitialLayout()
			elapsed = true
		}
	}

	return nil
}

// Scuffed version, placeholder till we have a decent layout system in place.
func (app *App) loadInitialLayout() {
	rect := app.contentContainer.GetWidget().Rect

	left, right := SplitRect(rect, "horizontal", 0.6, 0)
	leftA, rightA := SplitRect(right, "horizontal", 0.5, 0)
	topRightA, bottomRightA := SplitRect(rightA, "vertical", 0.5, 0)
	topLeftA, bottomLeftA := SplitRect(leftA, "vertical", 0.5, 0)

	// Change the name of the exchange that matches the one in cmd/main.go
	// pairBTC := event.NewPair("bybit", "btcusdt")
	// pairETH := event.NewPair("bybit", "ethusdt")
	pairBTC := event.NewPair("binancef", "btcusdt")
	pairETH := event.NewPair("binancef", "ethusdt")

	interval := int64(1)

	chartWidget := NewChartWidget(pairBTC, interval)
	chartWidget.AddLayer(NewHeatmapLayer(pairBTC))

	orderbookWidget := NewOrderbookWidget(pairBTC)
	tradesWidget := NewTradesWidget(pairBTC)

	orderbookWidgetEth := NewOrderbookWidget(pairETH)
	tradesWidgetEth := NewTradesWidget(pairETH)

	app.ui.AddWindow(NewWindow(chartWidget, "Aggregated heatmap", left))
	app.ui.AddWindow(NewWindow(orderbookWidget, "Orderbook TRUMP/USD", topRightA))
	app.ui.AddWindow(NewWindow(tradesWidget, "Trades TRUMP/USD", bottomRightA))

	app.ui.AddWindow(NewWindow(orderbookWidgetEth, "Orderbook ETH", topLeftA))
	app.ui.AddWindow(NewWindow(tradesWidgetEth, "Trades ETH", bottomLeftA))
}

func (app *App) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	panic("Market Monkey Terminal running with an unsupported Ebiten Engine version")
}

func (app *App) LayoutF(logicWidth, logicHeight float64) (float64, float64) {
	scale := ebiten.Monitor().DeviceScaleFactor()
	canvasWidth := math.Ceil(logicWidth * scale)
	canvasHeight := math.Ceil(logicHeight * scale)
	return canvasWidth, canvasHeight
}

// Temporary code here until the actual layout system is in place
func (app *App) getWidgetRect(size string) img.Rectangle {
	rect := app.contentContainer.GetWidget().Rect
	_, right := SplitRect(rect, "horizontal", 0.6, 0)
	_, rightA := SplitRect(right, "horizontal", 0.5, 0)

	switch size {
	case "small":
		topRightA, _ := SplitRect(rightA, "vertical", 0.5, 0)
		rect := img.Rect(0, 0, topRightA.Dx(), topRightA.Dy())
		return rect
	case "large":
		rect := img.Rect(0, 0, right.Dx(), right.Dy()/2)
		return rect
	default:
		panic("invalid widget type")
	}
}
