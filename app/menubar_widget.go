package app

import (
	"fmt"
	img "image"
	"image/color"
	"marketmonkey/event"
	"marketmonkey/settings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/colornames"
)

type MenuBarWidget struct {
	*widget.Container
}

func NewMenuBarWidget() *MenuBarWidget {
	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{Left: int(settings.PanelPadding)}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(0, int(settings.AppHeaderHeight)),
		),
	)

	innerContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.PanelBackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
		)),

		widget.ContainerOpts.WidgetOpts(
			// Make the toolbar fill the whole horizontal space of the screen.
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				StretchHorizontal:  true,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	chartButton := makeMenubarButton("Chart", "chart")
	orderbookButton := makeMenubarButton("Orderbook", "orderbook")
	tradesButton := makeMenubarButton("Trades", "trades")
	innerContainer.AddChild(
		chartButton,
		orderbookButton,
		tradesButton,
	)

	container.AddChild(innerContainer)

	return &MenuBarWidget{
		Container: container,
	}
}

func (w *MenuBarWidget) PreferredSize() (int, int) {
	return 0, int(settings.AppHeaderHeight)
}

func makeMenubarButton(name string, widgetType string) *widget.Button {
	exchangeButton := newToolbarButton(name)
	marketButtons := make([]*widget.Button, len(settings.Markets))
	i := 0
	for _, market := range settings.Markets {
		button := newToolbarMenuEntry(market.Name)
		marketButtons[i] = button
		syms := make([]*widget.Button, len(market.Symbols))
		for _, symbol := range market.Symbols {
			sym := newToolbarMenuEntry(symbol.Name)
			sym.ClickedEvent.AddHandler((func(args any) {
				pair := event.NewPair(market.Name, symbol.Name)
				windowName := fmt.Sprintf("%s %s", name, pair)
				switch widgetType {
				case "orderbook":
					orderbookWidget := NewOrderbookWidget(pair)
					app.ui.AddWindow(NewWindow(orderbookWidget, windowName, app.getWidgetRect("small")))
				case "trades":
					tradesWidget := NewTradesWidget(pair)
					app.ui.AddWindow(NewWindow(tradesWidget, windowName, app.getWidgetRect("small")))
				case "chart":
					chartWidget := NewChartWidget(pair, 1)
					chartWidget.AddLayer(NewHeatmapLayer(pair))
					app.ui.AddWindow(NewWindow(chartWidget, windowName, app.getWidgetRect("large")))
				}
			}))
			syms[i] = sym
			i++
		}
		button.ClickedEvent.AddHandler(func(args any) {
			openToolbarMenu(exchangeButton.GetWidget(), app.ui, syms...)
		})
	}
	exchangeButton.ClickedEvent.AddHandler(func(args any) {
		openToolbarMenu(exchangeButton.GetWidget(), app.ui, marketButtons...)
	})
	return exchangeButton
}

func newToolbarButton(label string) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    image.NewNineSliceColor(color.Transparent),
			Hover:   image.NewNineSliceColor(settings.MenuButtonHoverBg),
			Pressed: image.NewNineSliceColor(settings.MenuButtonClickBg),
		}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
		widget.ButtonOpts.Text(label, settings.FontSM, &widget.ButtonTextColor{
			Idle:     color.White,
			Disabled: colornames.Gray,
			Hover:    color.Black,
			Pressed:  color.Black,
		}),
		widget.ButtonOpts.TextPadding(widget.Insets{
			Top:    4,
			Left:   12,
			Right:  12,
			Bottom: 4,
		}),
	)
}

func newToolbarMenuEntry(label string) *widget.Button {
	// Create a button for a menu entry.
	return widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle:    image.NewNineSliceColor(color.Transparent),
			Hover:   image.NewNineSliceColor(settings.MenuButtonHoverBg),
			Pressed: image.NewNineSliceColor(colornames.White),
		}),
		widget.ButtonOpts.Text(label, settings.FontSM, &widget.ButtonTextColor{
			Idle:     color.White,
			Disabled: colornames.Gray,
			Hover:    color.Black,
			Pressed:  color.Black,
		}),
		widget.ButtonOpts.TextPosition(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.ButtonOpts.TextPadding(widget.Insets{Left: 16, Right: 16}),
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
		widget.ButtonOpts.TextPadding(widget.Insets{
			Top:    4,
			Left:   12,
			Right:  12,
			Bottom: 4,
		}),
	)
}

func openToolbarMenu(opener *widget.Widget, ui *ebitenui.UI, entries ...*widget.Button) {
	c := widget.NewContainer(
		// Set the background to a translucent black.
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(settings.BackgroundColor)),

		// Menu entries should be arranged vertically.
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(4),
				widget.RowLayoutOpts.Padding(widget.Insets{Top: 1, Bottom: 1}),
			),
		),

		// Set the minimum size for the menu.
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(64, 0)),
	)

	for _, entry := range entries {
		c.AddChild(entry)
	}

	w, h := c.PreferredSize()

	window := widget.NewWindow(
		// Set the menu to be a modal. This makes it block UI interactions to anything ese.
		widget.WindowOpts.Modal(),
		widget.WindowOpts.Contents(c),

		// Close the menu if the user clicks outside of it.
		widget.WindowOpts.CloseMode(widget.CLICK),

		// Position the menu below the menu button that it belongs to.
		widget.WindowOpts.Location(
			img.Rect(
				opener.Rect.Min.X,
				opener.Rect.Min.Y+opener.Rect.Max.Y,
				opener.Rect.Min.X+w,
				opener.Rect.Min.Y+opener.Rect.Max.Y+opener.Rect.Min.Y+h,
			),
		),
	)

	// Immediately add the menu to the UI.
	ui.AddWindow(window)
}
