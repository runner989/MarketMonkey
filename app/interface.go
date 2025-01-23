package app

import (
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type widgetCloser interface {
	widget.PreferredSizeLocateableWidget
	Close(*widget.WindowClosedEventArgs)
}

type Toolbar interface {
	Toolbar() *widget.Container
}

type layer interface {
	initialize(*ChartWidget)
	update(*ChartWidget)
	render(*ebiten.Image, *ChartWidget)
	delete()
}
