package app

import (
	"fmt"
	"image/color"
	"marketmonkey/settings"

	img "image"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/exp/shiny/materialdesign/colornames"
)

// func NewWindow(widg widgetCloser, title string, x, y, w, h int) *widget.Window {
func NewWindow(widg widgetCloser, title string, rect img.Rectangle) *widget.Window {
	content := CreateContainer(rect.Dx(), rect.Dy())
	content.AddChild(widg)

	container := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.BackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{
				Top:    int(2 * settings.Scale),
				Bottom: int(2 * settings.Scale),
				Right:  int(2 * settings.Scale),
				Left:   int(2 * settings.Scale),
			}),
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
			widget.RowLayoutOpts.Spacing(int(12 * settings.Scale)),
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Padding(widget.Insets{Left: int(settings.PanelPadding)}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				StretchHorizontal:  true,
				StretchVertical:    true,
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	innerContainer.AddChild(widget.NewText(
		widget.TextOpts.Text(title, settings.FontSM, color.NRGBA{254, 255, 255, 255}),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	))

	foo, ok := widg.(Toolbar)
	if ok {
		innerContainer.AddChild(foo.Toolbar())
	}
	container.AddChild(innerContainer)

	// Create the new window object. The window object is not tied to a container. Its location and
	// size are set manually using the SetLocation method on the window and added to the UI with ui.AddWindow()
	// Set the Button callback below to see how the window is added to the UI.
	window := widget.NewWindow(
		//Set the main contents of the window
		widget.WindowOpts.Contents(content),
		//Set the titlebar for the window (Optional)
		widget.WindowOpts.TitleBar(container, int(settings.PanelHeaderHeight)),
		//Set the window above everything else and block input elsewhere
		// widget.WindowOpts.Modal(),
		//Set how to close the window. CLICK_OUT will close the window when clicking anywhere
		//that is not a part of the window object
		//Indicates that the window is draggable. It must have a TitleBar for this to work
		widget.WindowOpts.ClosedHandler(widg.Close),
		widget.WindowOpts.Draggable(),
		//Set the window resizeable
		widget.WindowOpts.Resizeable(),
		//Set the minimum size the window can be
		widget.WindowOpts.MinSize(content.GetWidget().MinWidth, content.GetWidget().MinHeight),
		//Set the maximum size a window can be
		//widget.WindowOpts.MaxSize(300, 300),
		//Set the callback that triggers when a move is complete
		// widget.WindowOpts.MoveHandler(func(args *widget.WindowChangedEventArgs) {
		// 	//fmt.Printf("%+v\n", window.ddd)
		// }),
		//Set the callback that triggers when a resize is complete
		widget.WindowOpts.ResizeHandler(func(args *widget.WindowChangedEventArgs) {
			fmt.Println("Window Resized")
		}),
	)

	container.AddChild(windowCloseButton(window.Close))

	window.SetLocation(rect)

	return window
}

func windowCloseButton(onClose func()) *widget.Button {
	img := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(settings.ButtonIdleColor),
		Hover:   image.NewNineSliceColor(settings.ButtonHoverColor),
		Pressed: image.NewNineSliceColor(settings.ButtonPressedColor),
	}

	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(
			// instruct the container's anchor layout to center the button both horizontally and vertically
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding:            widget.Insets{Right: int(8 * settings.Scale)},
			}),
		),
		// specify the images to use
		widget.ButtonOpts.Image(img),

		// specify the button's text, the font face, and the color
		//widget.ButtonOpts.Text("Hello, World!", face, &widget.ButtonTextColor{
		widget.ButtonOpts.Text("x", settings.FontSM, &widget.ButtonTextColor{
			Idle:    colornames.White,
			Hover:   colornames.White,
			Pressed: colornames.White,
		}),
		//widget.ButtonOpts.TextProcessBBCode(true),
		// specify that the button's text needs some padding for correct display
		widget.ButtonOpts.TextPadding(widget.Insets{
			Left:  int(4 * settings.Scale),
			Right: int(4 * settings.Scale),
		}),
		widget.ButtonOpts.PressedHandler(func(args *widget.ButtonPressedEventArgs) {}),
		widget.ButtonOpts.ReleasedHandler(func(args *widget.ButtonReleasedEventArgs) {}),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			onClose()
		}),
		widget.ButtonOpts.CursorEnteredHandler(func(args *widget.ButtonHoverEventArgs) {}),
		widget.ButtonOpts.CursorMovedHandler(func(args *widget.ButtonHoverEventArgs) {}),
		widget.ButtonOpts.CursorExitedHandler(func(args *widget.ButtonHoverEventArgs) {}),
	)
}

func SetWindowLocation(window *widget.Window, x, y int) {
	w := window.MinSize.X
	h := window.MinSize.Y
	rect := img.Rect(0, 0, w, h)
	rect = rect.Add(img.Point{X: x, Y: y})
	window.SetLocation(rect)
}

func CreateContainer(w, h int) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(
			image.NewNineSliceColor(settings.BackgroundColor),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{
				Right: int(2 * settings.Scale),
				Left:  int(2 * settings.Scale),
				Bottom: int(2 * settings.Scale),
			}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(w, h),
		),
	)
}
