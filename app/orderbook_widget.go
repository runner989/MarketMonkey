package app

import (
	"fmt"
	"image/color"
	"marketmonkey/actor/session"
	"marketmonkey/event"
	"marketmonkey/settings"
	"slices"

	"github.com/anthdm/hollywood/actor"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type OrderbookWidget struct {
	*widget.Container

	pair       event.Pair
	streams    []session.Stream
	eventCh    chan any
	orderbook  event.Orderbook
	sessionPID *actor.PID

	rows []*OrderbookRow
}

func NewOrderbookWidget(pair event.Pair) *OrderbookWidget {
	eventCh := make(chan any)
	streams := []session.Stream{{
		Stream: event.StreamOrderbook,
	}}
	pid := app.engine.Spawn(session.New(eventCh, pair, streams), "session")

	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(int(10 * settings.Scale))),
			widget.RowLayoutOpts.Spacing(0),
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

	rows := []*OrderbookRow{}
	for i := 0; i < 14; i++ {
		r := NewOrderbookRow()
		container.AddChild(r)
		rows = append(rows, r)
	}

	panel := &OrderbookWidget{
		Container:  container,
		rows:       rows,
		pair:       pair,
		streams:    streams,
		eventCh:    eventCh,
		sessionPID: pid,
	}

	go panel.receiveData()

	return panel
}

func (p *OrderbookWidget) receiveData() {
	for ev := range p.eventCh {
		switch msg := ev.(type) {
		case event.Orderbook:
			p.orderbook = msg
		}
	}
}

func (p *OrderbookWidget) Render(screen *ebiten.Image) {
	p.Container.Render(screen)

	for i := len(p.orderbook.AskPrices) - 1; i >= 0; i-- {
		price := p.orderbook.AskPrices[i]
		size := p.orderbook.AskSizes[i]
		sum := p.orderbook.AskSums[i]
		p.rows[(7-1)-i].priceLabel.Label = fmt.Sprintf("%.2f", price)
		p.rows[(7-1)-i].priceLabel.Color = settings.Red
		p.rows[(7-1)-i].sizeLabel.Label = fmt.Sprintf("%.2f", size)
		p.rows[(7-1)-i].sumLabel.Label = fmt.Sprintf("%.2f", sum)

		label := p.rows[i].Container
		fillPerc := float32((sum / slices.Max(p.orderbook.AskSums)) * float64(label.GetWidget().Rect.Dx()))
		rect := label.GetWidget().Rect
		vector.DrawFilledRect(screen, float32(rect.Max.X)-fillPerc, float32(rect.Min.Y), fillPerc, float32(rect.Dy()), settings.OrderbookRed, false)
	}

	for i := 0; i < len(p.orderbook.BidPrices); i++ {
		price := p.orderbook.BidPrices[i]
		size := p.orderbook.BidSizes[i]
		sum := p.orderbook.BidSums[i]
		p.rows[i+7].priceLabel.Label = fmt.Sprintf("%.2f", price)
		p.rows[i+7].priceLabel.Color = settings.Green
		p.rows[i+7].sizeLabel.Label = fmt.Sprintf("%.2f", size)
		p.rows[i+7].sumLabel.Label = fmt.Sprintf("%.2f", sum)

		label := p.rows[i+7].Container
		fillPerc := float32((sum / slices.Max(p.orderbook.BidSums)) * float64(label.GetWidget().Rect.Dx()))

		rect := label.GetWidget().Rect
		vector.DrawFilledRect(screen, float32(rect.Max.X)-fillPerc, float32(rect.Min.Y), fillPerc, float32(rect.Dy()), settings.OrderbookGreen, false)
	}
}

func (p *OrderbookWidget) PreferredSize() (int, int) {
	return 0, 0
}

func (p *OrderbookWidget) GetWidget() *widget.Widget {
	return p.Container.GetWidget()
}

func (p *OrderbookWidget) Close(_ *widget.WindowClosedEventArgs) {
	app.engine.Poison(p.sessionPID)
}

type OrderbookRow struct {
	*widget.Container

	priceLabel *widget.Text
	sizeLabel  *widget.Text
	sumLabel   *widget.Text
}

func NewOrderbookRow() *OrderbookRow {
	priceLabel := widget.NewText(
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
		widget.TextOpts.Text("....", settings.FontSM, color.White),
	)
	sizeLabel := widget.NewText(
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
		widget.TextOpts.Text("44.44", settings.FontSM, color.White),
	)
	sumLabel := widget.NewText(
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
			}),
		),
		widget.TextOpts.Text("44.44", settings.FontSM, color.White),
	)

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart,
				Stretch:  true,
			}),
			widget.WidgetOpts.MinSize(0, int(24*settings.Scale)),
		),
	)

	root.AddChild(priceLabel, sizeLabel, sumLabel)
	return &OrderbookRow{
		Container:  root,
		priceLabel: priceLabel,
		sumLabel:   sumLabel,
		sizeLabel:  sizeLabel,
	}
}

func (r *OrderbookRow) Render(screen *ebiten.Image) {
	r.Container.Render(screen)
}

func (r *OrderbookRow) PreferredSize() (int, int) {
	return int(10 * settings.Scale), int(24 * settings.Scale)
}

func (r *OrderbookRow) GetWidget() *widget.Widget {
	return r.Container.GetWidget()
}
