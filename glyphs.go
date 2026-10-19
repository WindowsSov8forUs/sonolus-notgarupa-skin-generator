package skin

import (
	"fmt"
	"image/color"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/glyph"
)

// glyphScale is the supersampling of pre-rendered text relative to the
// original UI pixel size (1 UI unit ≈ 1.8 device px on common phones).
const glyphScale = 2.0

type glyphSet struct {
	size     float64
	hex      string
	chars    string
	spacingX float64
	use      string
}

var glyphSets = []glyphSet{
	{28, "BEBEBE", "0123456789", 1, "score leading zeros (advance + spacingX 1)"},
	{28, "FF3B72", "0123456789", 1, "score significant digits"},
	{18, "00C000", "0123456789/", 0, "life value (positive) and maximum"},
	{18, "FE2349", "0123456789", 0, "life value at zero"},
	{18, "505050", "/", 0, "life separator"},
	{12, "FFFFFF", "CBAS", 0, "score rank letters (SS = two S)"},
}

type textLabel struct {
	id, text string
	size     float64
	hex      string
	spacingX float64
	use      string
}

var textLabels = []textLabel{
	{"autolive", "オートライブ", 24, "FFFFFF", -1, "Auto Live caption"},
}

func parseHex(h string) color.NRGBA {
	var r, g, b uint8
	_, _ = fmt.Sscanf(h, "%02X%02X%02X", &r, &g, &b)
	return color.NRGBA{r, g, b, 255}
}

func addGlyphs(b *skinBuilder) error {
	data, err := apkFile("fonts/sgm/rank-label-font.ttf")
	if err != nil {
		return err
	}
	f, err := glyph.Parse(data)
	if err != nil {
		return err
	}
	add := func(name, text string, size float64, hex string, spacingX float64, kind, use string) error {
		g, err := f.Render(text, size*glyphScale, parseHex(hex), spacingX*glyphScale)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		w := float64(g.Image.Rect.Dx()) / glyphScale
		h := float64(g.Image.Rect.Dy()) / glyphScale
		return b.add(SpriteMeta{
			Name: name, Kind: kind, Group: "text", Source: "APK fonts/sgm/rank-label-font.ttf (sgm, TT-Shin Go)",
			W: int(w + 0.5), H: int(h + 0.5), Scale: glyphScale,
			Pivot: [2]float64{g.OffsetX / float64(g.Image.Rect.Dx()), g.Descent / (g.Ascent + g.Descent)},
			Glyph: &GlyphMeta{Font: "sgm", Size: size, Advance: g.Advance / glyphScale, Ascent: g.Ascent / glyphScale,
				Descent: g.Descent / glyphScale, OriginX: g.OffsetX / glyphScale, Text: text, SpacingX: spacingX},
			Note: use + "; pivot = pen origin on the baseline",
		}, g.Image)
	}
	for _, s := range glyphSets {
		for _, ch := range s.chars {
			name := fmt.Sprintf("sgm_%g_%s_%c", s.size, s.hex, ch)
			if err := add(name, string(ch), s.size, s.hex, 0, "glyph", s.use); err != nil {
				return err
			}
		}
	}
	for _, t := range textLabels {
		name := fmt.Sprintf("sgmtext_%g_%s_%s", t.size, t.hex, t.id)
		if err := add(name, t.text, t.size, t.hex, t.spacingX, "text", t.use); err != nil {
			return err
		}
	}
	return nil
}
