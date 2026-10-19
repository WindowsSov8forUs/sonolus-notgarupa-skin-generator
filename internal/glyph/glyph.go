// Package glyph pre-renders text with a TrueType/OpenType font, because a
// Sonolus engine cannot rasterise text at runtime.
package glyph

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Font struct{ f *opentype.Font }

func Parse(data []byte) (*Font, error) {
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	return &Font{f: f}, nil
}

// Glyph is one pre-rendered character cell.
//
// The image spans [0, Advance] horizontally and [descent, ascent] vertically,
// so the pen origin is at (0, Ascent) from the top-left corner of the image.
// Ink that overhangs the advance box is kept by widening the image; OffsetX is
// the pen origin's x inside the image.
type Glyph struct {
	Image   *image.NRGBA
	Advance float64 // pixels at the rendered size
	Ascent  float64
	Descent float64
	OffsetX float64
}

// Render draws a string of text (a single glyph or a whole label) at `size`
// pixels per em, coloured `c`, with extra `spacingX` pixels after every rune
// (NGUI spacingX semantics).
func (f *Font) Render(text string, size float64, c color.NRGBA, spacingX float64) (*Glyph, error) {
	face, err := opentype.NewFace(f.f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	defer face.Close()
	m := face.Metrics()
	ascent := float64(m.Ascent) / 64
	descent := float64(m.Descent) / 64

	// Measure ink bounds and advance.
	var pen fixed.Int26_6
	minX, maxX := 0.0, 0.0
	prev := rune(-1)
	for _, r := range text {
		if prev >= 0 {
			pen += face.Kern(prev, r)
		}
		b, adv, ok := face.GlyphBounds(r)
		if !ok {
			return nil, fmt.Errorf("font has no glyph for %q", r)
		}
		minX = math.Min(minX, float64(pen+b.Min.X)/64)
		maxX = math.Max(maxX, float64(pen+b.Max.X)/64)
		pen += adv + fixed.Int26_6(spacingX*64)
		prev = r
	}
	advance := float64(pen) / 64
	left := math.Floor(math.Min(0, minX)) - 1
	right := math.Ceil(math.Max(advance, maxX)) + 1
	w := int(right - left)
	h := int(math.Ceil(ascent + descent))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(-left * 64), Y: fixed.Int26_6(ascent * 64)},
	}
	prev = -1
	for _, r := range text {
		if prev >= 0 {
			d.Dot.X += face.Kern(prev, r)
		}
		d.DrawString(string(r))
		d.Dot.X += fixed.Int26_6(spacingX * 64)
		prev = r
	}
	return &Glyph{Image: img, Advance: advance, Ascent: ascent, Descent: descent, OffsetX: -left}, nil
}
