package app

import (
	"fmt"
	"marketmonkey/settings"
	"math"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type PriceScaleWidget struct {
	*widget.Container

	chart *ChartWidget
}

func NewPriceScale(chart *ChartWidget) *PriceScaleWidget {
	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{Left: 60}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(settings.ChartPriceScaleWidth), 0),
		),
	)
	return &PriceScaleWidget{
		Container: container,
		chart:     chart,
	}
}

func (p *PriceScaleWidget) Render(screen *ebiten.Image) {
	p.Container.Render(screen)

	// Multiply the result of getDynamicPriceStep by 2 (or 5, etc.)
	stepSize := getDynamicPriceStep(p.chart.visiblePriceRange) * 2
	startPrice := math.Floor(p.chart.minPrice/stepSize) * stepSize
	endPrice := math.Ceil(p.chart.maxPrice/stepSize) * stepSize

	rect := p.Container.GetWidget().Rect
	for price := startPrice; price <= endPrice; price += stepSize {
		y := p.chart.getPriceYScreen(price)
		if y > float32(rect.Min.Y) && y < float32(rect.Min.Y+rect.Dy()) {
			label := fmt.Sprintf("%.2f", price)
			op := text.DrawOptions{}
			font := settings.FontSM
			fh := font.Metrics().CapHeight
			op.GeoM.Translate(float64(rect.Min.X)+float64(settings.ChartPriceScaleMargin), float64(y)-fh)
			text.Draw(screen, label, font, &op)
		}
	}
}
