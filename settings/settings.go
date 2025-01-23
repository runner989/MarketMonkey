package settings

import (
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/exp/shiny/materialdesign/colornames"
)

var (
	Scale = float32(ebiten.Monitor().DeviceScaleFactor())

	// Colors
	Black          = color.RGBA{12, 14, 17, 255}
	Red            = color.RGBA{246, 71, 93, 255}
	Green          = color.RGBA{45, 189, 133, 255}
	OrderbookRed   = color.RGBA{52, 30, 39, 1}
	OrderbookGreen = color.RGBA{27, 45, 43, 1}

	// APP
	BackgroundColor  = Black
	BackgroundColor2 = color.RGBA{23, 26, 32, 255}
	BackgroundColor3 = color.RGBA{23, 26, 32, 0}

	// App header
	AppHeaderHeight          = 40 * Scale
	AppHeaderBackGroundColor = BackgroundColor2

	// App footer
	AppFooterHeight          = 24 * Scale
	AppFooterBackGroundColor = BackgroundColor2

	// App divider line color
	DividerColor  = Black
	DividerHeight = 3 * Scale

	PanelBackgroundColor = BackgroundColor2
	PanelDividerColor    = color.RGBA{52, 59, 71, 255}
	PanelHeaderHeight    = 40 * Scale
	PanelPadding         = 12 * Scale

	ButtonIdleColor    = color.RGBA{42, 49, 57, 1}
	ButtonHoverColor   = color.RGBA{42, 49, 57, 100}
	ButtonPressedColor = color.RGBA{42, 49, 57, 200}

	ChartPriceScaleWidth  = 80 * Scale
	ChartTimeScaleHeight  = 30 * Scale
	ChartPriceScaleMargin = 12 * Scale
	// The buffer around the center price when the chart is initialized
	ChartPriceScaleDefaultPriceRange         = 100
	ChartRenderPriceLine                     = true
	ChartPriceLabelWidth                     = ChartPriceScaleWidth
	ChartPriceLabelHeight                    = 20 * Scale
	ChartPriceLabelColor                     = color.RGBA{45, 189, 133, 255}
	ChartCrossHairColor                      = colornames.BlueGrey800
	LineChartLineColor                       = colornames.Blue300
	LineChartStrokeWidth                     = 1.5 * Scale
	CandleStickGreen                         = Green
	CandleStickRed                           = Red
	VolumeBarGreen                           = colornames.GreenA100
	VolumeBarRed                             = colornames.PinkA100
	VolumeBarHeightPerc              float32 = 0.15

	HeatmapStartColor = PanelBackgroundColor
	HeatmapEndColor   = color.RGBA{255, 255, 0, 255}

	FontSM   text.Face
	FontBase text.Face

	FlashLastTrade      = true
	FlashLastTradeColor = colornames.White

	// Buttons
	PanChartButton = ebiten.MouseButton0

	MenuButtonHoverBg         = colornames.Orange300
	MenuButtonClickBg         = colornames.Orange600
	MenuButtonActiveBg        = colornames.Orange600
	MenuButtonTextColorIdle   = colornames.White
	MenuButtonTextColorActive = colornames.Orange600

	HeatmapSizeTextWidth  float64
	HeatmapSizeTextHeight float64

	// Theming for now. Will add the other colors in here later.
	ColorPrimary        = colornames.Orange300
	ColorPrimaryLighter = colornames.Orange100
	ColorPrimaryDarker  = colornames.Orange600

	TickIntervals = []IntervalConfig{
		{Interval: 1},
		{Interval: 5},
		{Interval: 60},
		{Interval: 300},
		{Interval: 900},
		{Interval: 3600},
		{Interval: 86400},
		{Interval: 604800},
		{Interval: 2629800},
	}
)

type IntervalConfig struct {
	Interval int64
	Disabled bool
}

func init() {
	FontSM, _ = LoadFont(12)
	FontBase, _ = LoadFont(13)

	HeatmapSizeTextWidth, HeatmapSizeTextHeight = text.Measure("12.00", FontSM, FontSM.Metrics().VLineGap)
}

func LoadFont(size float64) (text.Face, error) {
	b, err := os.Open("assets/jetbrains.ttf")
	if err != nil {
		return nil, err
	}
	s, err := text.NewGoTextFaceSource(b)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &text.GoTextFace{
		Source: s,
		Size:   size * ebiten.Monitor().DeviceScaleFactor(),
	}, nil
}
