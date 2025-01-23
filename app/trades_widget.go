package app

import (
	"fmt"
	"image/color"
	"marketmonkey/actor/session"
	"marketmonkey/event"
	"marketmonkey/settings"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type TradesWidget struct {
	*widget.Container

	pair       event.Pair
	eventCh    chan any
	sessionPID *actor.PID
	trades     []event.Trade
	rows       []*TradeRow
}

func NewTradesWidget(pair event.Pair) *TradesWidget {
	eventCh := make(chan any)
	streams := []session.Stream{{
		Stream: event.StreamTrades,
	}}
	pid := app.engine.Spawn(session.New(eventCh, pair, streams), "session")

	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(int(10*settings.Scale))),
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

	rows := []*TradeRow{}
	for i := 0; i < 16; i++ {
		r := NewTradeRow()
		container.AddChild(r)
		rows = append(rows, r)
	}

	widget := &TradesWidget{
		Container:  container,
		sessionPID: pid,
		pair:       pair,
		trades:     []event.Trade{},
		rows:       rows,
		eventCh:    eventCh,
	}

	go widget.receiveData()

	return widget
}

func (t *TradesWidget) receiveData() {
	for ev := range t.eventCh {
		switch msg := ev.(type) {
		case event.Trade:
			for i := len(t.rows) - 1; i > 0; i-- {
				t.rows[i].priceLabel.Label = t.rows[i-1].priceLabel.Label
				t.rows[i].priceLabel.Color = t.rows[i-1].priceLabel.Color
				t.rows[i].sizeLabel.Label = t.rows[i-1].sizeLabel.Label
				t.rows[i].timeLabel.Label = t.rows[i-1].timeLabel.Label
			}
			color := settings.Red
			if msg.IsBuy {
				color = settings.Green
			}
			t.rows[0].priceLabel.Label = fmt.Sprintf("%.2f", msg.Price)
			t.rows[0].priceLabel.Color = color
			t.rows[0].sizeLabel.Label = fmt.Sprintf("%.2f", msg.Qty)
			t.rows[0].timeLabel.Label = time.UnixMilli(msg.Unix).Format("15:04:05")
			t.rows[0].flash = true
		}
	}
}

func (w *TradesWidget) Render(screen *ebiten.Image) {
	w.Container.Render(screen)
}

func (w *TradesWidget) Update() {
	w.Container.Update()
}

func (*TradesWidget) PreferredSize() (int, int) {
	return 0, 0
}

func (w *TradesWidget) GetWidget() *widget.Widget {
	return w.Container.GetWidget()
}

func (w *TradesWidget) Close(_ *widget.WindowClosedEventArgs) {
	app.engine.Poison(w.sessionPID)
}

type TradeRow struct {
	*widget.Container

	priceLabel *widget.Text
	sizeLabel  *widget.Text
	timeLabel  *widget.Text
	image      *ebiten.Image
	flash      bool
}

func NewTradeRow() *TradeRow {
	priceLabel := widget.NewText(
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
		widget.TextOpts.Text(".....", settings.FontSM, color.White),
	)
	sizeLabel := widget.NewText(
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
		widget.TextOpts.Text(".....", settings.FontSM, color.White),
	)
	timeLabel := widget.NewText(
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
			}),
		),
		widget.TextOpts.Text(time.Now().Format("15:04:05"), settings.FontSM, color.White),
	)
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart,
				Stretch:  true,
			}),
			widget.WidgetOpts.MinSize(0, int(24*settings.Scale)),
		),
	)
	container.AddChild(priceLabel, sizeLabel, timeLabel)
	image := ebiten.NewImage(1, 1)
	image.Fill(settings.FlashLastTradeColor)
	return &TradeRow{
		Container:  container,
		priceLabel: priceLabel,
		sizeLabel:  sizeLabel,
		timeLabel:  timeLabel,
		image:      image,
	}
}

func (r *TradeRow) Render(screen *ebiten.Image) {
	r.Container.Render(screen)
	rect := r.Container.GetWidget().Rect

	if settings.FlashLastTrade && r.flash {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(rect.Dx()), float64(rect.Dy()))
		op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
		op.Blend = ebiten.BlendSourceOver
		screen.DrawImage(r.image, op)
		r.flash = false
	}
}

func (r *TradeRow) PreferredSize() (int, int) {
	return int(10 * settings.Scale), int(24 * settings.Scale)
}

func (r *TradeRow) GetWidget() *widget.Widget {
	return r.Container.GetWidget()
}
