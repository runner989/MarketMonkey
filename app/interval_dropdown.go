package app

import (
	"image/color"

	"marketmonkey/settings"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func intervalDropdown(selectFn func(int64)) *widget.ListComboButton {
	enabledEntries := []any{}
	for _, entry := range settings.TickIntervals {
		if !entry.Disabled {
			enabledEntries = append(enabledEntries, entry.Interval)
		}
	}
	comboBox := widget.NewListComboButton(
		widget.ListComboButtonOpts.SelectComboButtonOpts(
			widget.SelectComboButtonOpts.ComboButtonOpts(
				widget.ComboButtonOpts.MaxContentHeight(300),
				widget.ComboButtonOpts.ButtonOpts(
					widget.ButtonOpts.Image(&widget.ButtonImage{
						Idle:     image.NewNineSliceColor(color.Transparent),
						Hover:    image.NewNineSliceColor(settings.ColorPrimary),
						Pressed:  image.NewNineSliceColor(settings.ColorPrimaryDarker),
						Disabled: image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
					}),
					// widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(0)),
					widget.ButtonOpts.Text("", settings.FontSM, &widget.ButtonTextColor{
						Idle:     color.White,
						Hover:    color.Black,
						Disabled: color.White,
					}),
					widget.ButtonOpts.TextPadding(widget.Insets{
						Top:    4,
						Left:   12,
						Right:  12,
						Bottom: 4,
					}),
					widget.ButtonOpts.WidgetOpts(
						widget.WidgetOpts.MinSize(0, 0),
						widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
							HorizontalPosition: widget.AnchorLayoutPositionCenter,
							VerticalPosition:   widget.AnchorLayoutPositionCenter,
						})),
				),
			),
		),
		widget.ListComboButtonOpts.ListOpts(
			widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(0, 0))),
			widget.ListOpts.Entries(enabledEntries),
			widget.ListOpts.ScrollContainerOpts(
				widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
					Idle: image.NewNineSliceColor(settings.BackgroundColor),
					Mask: image.NewNineSliceColor(settings.Black),
				}),
				widget.ScrollContainerOpts.Padding(widget.NewInsetsSimple(12)),
			),
			widget.ListOpts.SliderOpts(
				// This supposed to be the scrollbar, keeping it in for now for future reference
				widget.SliderOpts.Images(&widget.SliderTrackImage{
					Idle:  image.NewNineSliceColor(settings.Green),
					Hover: image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				}, &widget.ButtonImage{
					Idle:     image.NewNineSliceColor(settings.Green),
					Hover:    image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
					Pressed:  image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
					Disabled: image.NewNineSliceColor(color.NRGBA{100, 100, 100, 255}),
				}),
				widget.SliderOpts.MinHandleSize(0),
				widget.SliderOpts.TrackPadding(widget.NewInsetsSimple(0))),
			widget.ListOpts.EntryFontFace(settings.FontSM),
			widget.ListOpts.EntryColor(&widget.ListEntryColor{
				Selected:                   settings.Black,
				Unselected:                 color.White,
				SelectingBackground:        settings.ColorPrimaryDarker,
				SelectingFocusedBackground: settings.Black,
				SelectedBackground:         settings.ColorPrimaryLighter,
				SelectedFocusedBackground:  settings.ColorPrimary,
				FocusedBackground:          settings.ColorPrimary,
				DisabledUnselected:         color.NRGBA{100, 100, 100, 255}, // Foreground color for the disabled unselected entry
				DisabledSelected:           color.NRGBA{100, 100, 100, 255}, // Foreground color for the disabled selected entry
				DisabledSelectedBackground: color.NRGBA{100, 100, 100, 255}, // Background color for the disabled selected entry
			}),
			widget.ListOpts.EntryTextPadding(widget.NewInsetsSimple(5)),
		),
		// Define how the entry is displayed
		widget.ListComboButtonOpts.EntryLabelFunc(
			func(e any) string {
				return TickInterval(e.(int64)).String()
			},
			func(e any) string {
				return TickInterval(e.(int64)).String()
			}),
		widget.ListComboButtonOpts.EntrySelectedHandler(func(args *widget.ListComboButtonEntrySelectedEventArgs) {
			selectFn(args.Entry.(int64))
		}),
	)
	comboBox.SetSelectedEntry(settings.TickIntervals[0].Interval)

	return comboBox
}
