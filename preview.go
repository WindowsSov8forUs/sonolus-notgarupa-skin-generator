package skin

import (
	"image"
	"image/color"
	"math"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
)

// Preview sprites. The original has no chart preview; Sonolus preview draws
// the chart as flat columns, so the lane image is unprojected into a flat
// strip and the grid lines get plain fills.
const (
	previewLaneSprite = "notgarupa/preview/lane"
	previewLineSprite = "notgarupa/preview/line"
)

// Field geometry of the original (RhythmGameLines-local UI units of the
// 1160×610 bg_line_rhythm, bottom edge on the judge line): lane spacing at the
// judge line 2.2·S and the spawn row 0.95·8.870001·S above it, with S / UI unit
// = 1334 / (2·9.578571) for every aspect. A point h units above the judge line
// is scaled by k(h) = 1 − 0.95·h / fieldSpawnH toward the centre (the note
// motion: launch rate 0.05 at the spawn row).
const (
	fieldWidthPx   = 1160.0
	fieldS         = 1334 / (2 * 9.578571319580078)
	fieldLanePx    = 2.2 * fieldS
	fieldSpawnH    = 0.95 * 8.870001 * fieldS
	previewStripW  = 580 // half resolution: the strip spans ~0.8 screen units
	previewStripH  = 2
	previewRowsMin = 12  // skips the soft bottom edge
	previewRowsMax = 400 // k ≥ 0.35: above it the art fades into the haze
)

// flattenField unprojects a bg_line_rhythm into a previewStripW × previewStripH
// strip: each output column is the premultiplied average, over the rows
// [previewRowsMin, previewRowsMax] above the judge line, of the texel on the
// same field line (x scaled by k(h) toward the centre).
func flattenField(field *image.NRGBA) *image.NRGBA {
	w, h := field.Rect.Dx(), field.Rect.Dy()
	sx := float64(w) / fieldWidthPx // texture pixels per UI unit
	sy := float64(h) / 610
	out := image.NewNRGBA(image.Rect(0, 0, previewStripW, previewStripH))
	for x := 0; x < previewStripW; x++ {
		var acc [4]float64
		n := 0.0
		// two source columns per output column (box filter)
		for sub := 0.0; sub < 2; sub++ {
			u := (float64(x)*2 + sub + 0.5) / (previewStripW * 2) * fieldWidthPx // UI units from the left edge
			for hp := previewRowsMin; hp <= previewRowsMax; hp++ {
				k := 1 - 0.95*float64(hp)/fieldSpawnH
				fx := (fieldWidthPx/2+(u-fieldWidthPx/2)*k)*sx - 0.5
				fy := (610-float64(hp)-0.5)*sy - 0.5
				c := bilinear(field, fx, fy)
				a := c[3]
				acc[0] += c[0] * a
				acc[1] += c[1] * a
				acc[2] += c[2] * a
				acc[3] += a
				n++
			}
		}
		px := color.NRGBA{}
		if acc[3] > 0 {
			px = color.NRGBA{to8(acc[0] / acc[3]), to8(acc[1] / acc[3]), to8(acc[2] / acc[3]), to8(acc[3] / n)}
		}
		for y := 0; y < previewStripH; y++ {
			out.SetNRGBA(x, y, px)
		}
	}
	return out
}

// bilinear samples a straight-alpha image at pixel-centre coordinates (x, y),
// interpolating premultiplied colour; it returns straight RGBA in 0..1.
func bilinear(img *image.NRGBA, x, y float64) [4]float64 {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	var acc [4]float64
	for _, t := range [4][3]float64{{0, 0, (1 - fx) * (1 - fy)}, {1, 0, fx * (1 - fy)}, {0, 1, (1 - fx) * fy}, {1, 1, fx * fy}} {
		px := int(math.Min(math.Max(x0+t[0], 0), float64(w-1)))
		py := int(math.Min(math.Max(y0+t[1], 0), float64(h-1)))
		p := img.Pix[img.PixOffset(px, py):]
		a := float64(p[3]) / 255
		acc[0] += float64(p[0]) / 255 * a * t[2]
		acc[1] += float64(p[1]) / 255 * a * t[2]
		acc[2] += float64(p[2]) / 255 * a * t[2]
		acc[3] += a * t[2]
	}
	if acc[3] <= 0 {
		return [4]float64{}
	}
	return [4]float64{acc[0] / acc[3], acc[1] / acc[3], acc[2] / acc[3], acc[3]}
}

func to8(v float64) uint8 { return uint8(math.Round(math.Min(math.Max(v, 0), 1) * 255)) }

// addPreviewLane adds name: the flat preview strip of a lane image.
func addPreviewLane(b *skinBuilder, name, group, source string, field *image.NRGBA) error {
	return b.add(SpriteMeta{Name: name, Kind: "texture", Group: group, Source: "derived from " + source,
		Pivot: [2]float64{0.5, 0.5},
		Note:  "preview lane: bg_line_rhythm unprojected along the field perspective (k = 1 − 0.95·h/586.8 UI units) and averaged over 12–400 units above the judge line; spans the full 1160-unit field width = 7.572 lanes (lane 153.2 units), stretched along the chart time"},
		flattenField(field))
}

// previewLines are the tinted preview line fills (the beat lines use the
// white fill itself): the BPM change line, dark red.
var previewLines = []struct {
	c    [4]float64
	note string
}{
	{[4]float64{0xB4 / 255.0, 0x1E / 255.0, 0x28 / 255.0, 1}, "preview BPM change line"},
}

func addPreviewLines(b *skinBuilder) error {
	if err := b.add(SpriteMeta{Name: previewLineSprite, Kind: "sprite", Group: "preview", Source: "generated",
		Pivot: [2]float64{0.5, 0.5}, Note: "preview beat / measure line, uniform white fill"}, imgutil.Solid(4, 4, color.NRGBA{255, 255, 255, 255})); err != nil {
		return err
	}
	for _, l := range previewLines {
		if _, err := b.addTint(previewLineSprite, l.c, l.note); err != nil {
			return err
		}
	}
	return nil
}
