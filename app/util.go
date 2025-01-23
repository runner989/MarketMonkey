package app

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawText(screen *ebiten.Image, str string, font text.Face, x, y float64, color color.Color) {
	ops := text.DrawOptions{}
	ops.GeoM.Translate(x, y)
	ops.ColorScale.ScaleWithColor(color)
	text.Draw(screen, str, font, &ops)
}

// DrawDashedLine draws a dashed line from (x0,y0) to (x1,y1) on the given dst image.
func DrawDashedLine(
	dst *ebiten.Image,
	x0, y0, x1, y1 float32,
	thickness float32,
	dashLen, gapLen float32,
	clr color.Color,
	additive bool,
) {
	// Compute the total length of the line.
	totalLen := float64(math.Hypot(float64(x1-x0), float64(y1-y0)))
	if totalLen == 0 {
		return // Nothing to draw for zero-length line
	}

	// Calculate the direction vector for the line.
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	// Normalize direction.
	dirX := dx / totalLen
	dirY := dy / totalLen

	// Current drawing position starting at (x0, y0).
	currX := float64(x0)
	currY := float64(y0)

	// While there's remaining distance on the line.
	distanceDrawn := 0.0

	for distanceDrawn < totalLen {
		// Determine end of this dash segment.
		dashEnd := distanceDrawn + float64(dashLen)
		if dashEnd > totalLen {
			dashEnd = totalLen
		}

		// Calculate end point of this dash segment.
		endX := float32(currX + dirX*(dashEnd-distanceDrawn))
		endY := float32(currY + dirY*(dashEnd-distanceDrawn))

		// Draw the dash segment using vector.StrokeLine.
		vector.StrokeLine(
			dst,
			float32(currX), float32(currY),
			endX, endY,
			thickness,
			clr,
			additive,
		)

		// Update the distance drawn to the end of this dash.
		distanceDrawn = dashEnd

		// Skip the gap portion.
		distanceDrawn += float64(gapLen)
		if distanceDrawn > totalLen {
			break // No more space for additional dashes.
		}

		// Update current drawing position after the gap.
		currX = float64(x0) + dirX*distanceDrawn
		currY = float64(y0) + dirY*distanceDrawn
	}
}

// Simple HSL-based gradient. Hue shifts from 240° (blue) down to 0° (red).
// This often looks smoother than a random set of stops.
func GetIntensityColorNew(intensity float32) color.RGBA {
	// Clamp intensity to [0..1]
	if intensity < 0 {
		intensity = 0
	} else if intensity > 1 {
		intensity = 1
	}

	hueStart := 60.0
	hueEnd := 30.0
	// Hue range: 240 (blue) → 0 (red)
	hue := hueStart + (hueEnd-hueStart)*float64(intensity)
	sat := 1.0 // keep full saturation
	lum := 0.5 // mid luminance

	r, g, b := hslToRgb(hue, sat, lum)

	// Optionally fade alpha by intensity if you want more transparency at lower intensities:
	alphaF := math.Pow(float64(intensity), 2.0) // fade curve
	a := uint8(255.0 * alphaF)

	// Premultiply RGB by alpha factor for more consistent blending
	rF := float64(r) * alphaF
	gF := float64(g) * alphaF
	bF := float64(b) * alphaF

	return color.RGBA{uint8(rF), uint8(gF), uint8(bF), a}
}

// Convert HSL → RGB (0..255). Simple formula version.
func hslToRgb(h, s, l float64) (float64, float64, float64) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return 255.0 * (r + m), 255.0 * (g + m), 255.0 * (b + m)
}

var heatmapGradient = []struct {
	Stop    float32
	R, G, B uint8
}{
	{0.00, 0x1D, 0x0B, 0x39}, // Deep purple
	{0.70, 0x4B, 0x23, 0x8A}, // Purple
	{0.90, 0x83, 0xC3, 0x3F}, // Lime green
	{1, 0xFF, 0xD6, 0x00},    // Bright yellow
}

func GetIntensityColor(intensity float32) color.RGBA {
	// Clamp intensity to [0..1]
	if intensity < 0 {
		intensity = 0
	} else if intensity > 1 {
		intensity = 1
	}

	// 1) Find the base RGB from the gradient (fully saturated color).
	var baseR, baseG, baseB uint8
	for i := 0; i < len(heatmapGradient)-1; i++ {
		curr := heatmapGradient[i]
		next := heatmapGradient[i+1]

		if intensity >= curr.Stop && intensity <= next.Stop {
			// Fraction of the way between these stops:
			f := (intensity - curr.Stop) / (next.Stop - curr.Stop)

			// Linear interpolation:
			baseR = uint8(float32(curr.R) + f*float32(next.R-curr.R))
			baseG = uint8(float32(curr.G) + f*float32(next.G-curr.G))
			baseB = uint8(float32(curr.B) + f*float32(next.B-curr.B))
			break
		}
	}
	// If intensity=1 or something else, default to last color in the gradient
	if intensity == 1 {
		last := heatmapGradient[len(heatmapGradient)-1]
		baseR, baseG, baseB = last.R, last.G, last.B
	}

	// 2) Decide how strongly to fade out low intensities.
	//    A simple approach: alpha = intensity^power * 255
	//    e.g. intensity^1.0 is a linear fade, intensity^2.0 fades more aggressively, etc.
	fadePower := 1.0
	alphaF := float32(math.Pow(float64(intensity), fadePower))
	alpha8 := uint8(255 * alphaF)

	// 3) Premultiply the RGB by alpha, otherwise partially‐transparent blocks
	//    can still leave “ghosts” if R/G/B stay high while A is low.
	rF := float32(baseR) * alphaF
	gF := float32(baseG) * alphaF
	bF := float32(baseB) * alphaF

	return color.RGBA{
		R: uint8(rF),
		G: uint8(gF),
		B: uint8(bF),
		A: alpha8,
	}
}

func SplitRect(rect image.Rectangle, orientation string, ratio float64, spacing int) (pane1, pane2 image.Rectangle) {
	switch orientation {
	case "horizontal":
		totalW := rect.Dx()
		paneW := int(ratio * float64(totalW))

		pane1 = image.Rect(
			rect.Min.X,
			rect.Min.Y,
			rect.Min.X+paneW,
			rect.Max.Y,
		)

		pane2 = image.Rect(
			rect.Min.X+paneW+spacing,
			rect.Min.Y,
			rect.Max.X,
			rect.Max.Y,
		)

	case "vertical":
		totalH := rect.Dy()
		paneH := int(ratio * float64(totalH))

		pane1 = image.Rect(
			rect.Min.X,
			rect.Min.Y,
			rect.Max.X,
			rect.Min.Y+paneH,
		)

		pane2 = image.Rect(
			rect.Min.X,
			rect.Min.Y+paneH+spacing,
			rect.Max.X,
			rect.Max.Y,
		)

	default:
		pane1 = rect
		pane2 = image.Rectangle{}
	}

	return
}
