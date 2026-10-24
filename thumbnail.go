package skin

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
	xdraw "golang.org/x/image/draw"
)

// Item thumbnails: GarupaEditor's skin thumbnail composition
// (src/skin/thumbnails/skinThumbnailRenderer.ts), ported: the lane image
// stretched to a trapezoid, the judge line on its bottom edge and the
// centre-lane note (note_normal_3) above, drawn at 4× with a dark outline and
// scaled to 128×128. The note skin and the lane decide the image; the
// directional skin does not appear.
const (
	thumbS      = 4
	thumbSize   = 128 * thumbS
	thumbTop    = 23 * thumbS
	thumbBottom = 92 * thumbS
	thumbWidth  = 112 * thumbS
)

// thumbnail returns the PNG thumbnail of a note skin and a field bundle
// (fieldBundle is the full bundle path; note is the note_normal_3 sprite),
// composed once per Generate call.
func (ps *parts) thumbnail(key, fieldBundle string, note *image.NRGBA) ([]byte, error) {
	ps.mu.Lock()
	e, ok := ps.cache["thumbnail "+key]
	if !ok {
		e = &partEntry{}
		ps.cache["thumbnail "+key] = e
	}
	ps.mu.Unlock()
	e.once.Do(func() {
		_, tex, err := bundleSprites(ps.bd, fieldBundle)
		if err != nil {
			e.err = err
			return
		}
		field, err := tex("bg_line_rhythm.png")
		if err != nil {
			e.err = err
			return
		}
		judge, err := tex("game_play_line.png")
		if err != nil {
			e.err = err
			return
		}
		img, err := composeThumbnail(field, judge, note, fieldBundle == "ingameskin/fieldskin/skin_gbp2020")
		if err != nil {
			e.err = fmt.Errorf("thumbnail %s: %w", key, err)
			return
		}
		png, err := imgutil.EncodePNG(img)
		e.err = err
		e.p = &part{thumbnail: png}
	})
	if e.err != nil {
		return nil, e.err
	}
	return e.p.thumbnail, nil
}

func newRGBA(w, h int) *image.RGBA { return image.NewRGBA(image.Rect(0, 0, w, h)) }

// scaleOver draws src's rect sr scaled into dst's rect dr (source-over,
// high-quality resampling as the canvas does).
func scaleOver(dst draw.Image, dr image.Rectangle, src image.Image, sr image.Rectangle) {
	xdraw.CatmullRom.Scale(dst, dr, src, sr, xdraw.Over, nil)
}

// alphaBounds is the bounding box of the pixels with alpha ≥ threshold in
// rows fromY… (the canvas getImageData values, straight alpha).
func alphaBounds(img *image.NRGBA, threshold uint8, fromY int) (image.Rectangle, error) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	left, top, right, bottom := w, h, 0, 0
	for y := fromY; y < h; y++ {
		for x := 0; x < w; x++ {
			if img.Pix[img.PixOffset(x, y)+3] < threshold {
				continue
			}
			left, right = min(left, x), max(right, x+1)
			top, bottom = min(top, y), max(bottom, y+1)
		}
	}
	if right <= left {
		return image.Rectangle{}, fmt.Errorf("empty source image")
	}
	return image.Rect(left, top, right, bottom), nil
}

// isolateLaneBody keeps the continuous lane trapezoid of a field image with
// detached decorations (skin_gbp2020's stars): the straight edges are taken
// from two interior rows.
func isolateLaneBody(img *image.NRGBA) error {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	alpha := func(x, y int) uint8 { return img.Pix[img.PixOffset(x, y)+3] }
	span := func(y int) (float64, float64, error) {
		c := w / 2
		if alpha(c, y) < 16 {
			return 0, 0, fmt.Errorf("lane body not found")
		}
		l, r := c, c
		for l > 0 && alpha(l-1, y) >= 16 {
			l--
		}
		for r+1 < w && alpha(r+1, y) >= 16 {
			r++
		}
		return float64(l), float64(r), nil
	}
	y1, y2 := int(math.Round(float64(h)*.3)), int(math.Round(float64(h)*.75))
	al, ar, err := span(y1)
	if err != nil {
		return err
	}
	bl, br, err := span(y2)
	if err != nil {
		return err
	}
	for y := 0; y < h; y++ {
		t := float64(y-y1) / float64(y2-y1)
		l, r := al+(bl-al)*t, ar+(br-ar)*t
		for x := 0; x < w; x++ {
			if float64(x) < l-1 || float64(x) > r+1 {
				img.Pix[img.PixOffset(x, y)+3] = 0
			}
		}
	}
	return nil
}

// dilate is the max of alpha over a disk of radius (horizontal maxima per
// half-width, then rows).
func dilate(alpha []uint8, w, h, radius int) []uint8 {
	horizontal := [][]uint8{alpha}
	for r := 1; r <= radius; r++ {
		prev, row := horizontal[r-1], make([]uint8, len(alpha))
		copy(row, prev)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i := y*w + x
				if x >= r {
					row[i] = max(row[i], alpha[i-r])
				}
				if x+r < w {
					row[i] = max(row[i], alpha[i+r])
				}
			}
		}
		horizontal = append(horizontal, row)
	}
	out := make([]uint8, len(alpha))
	copy(out, alpha)
	for dy := -radius; dy <= radius; dy++ {
		row := horizontal[int(math.Floor(math.Sqrt(float64(radius*radius-dy*dy))))]
		for y := max(0, -dy); y < min(h, h-dy); y++ {
			t, s := y*w, (y+dy)*w
			for x := 0; x < w; x++ {
				out[t+x] = max(out[t+x], row[s+x])
			}
		}
	}
	return out
}

// thickened grows img by a disk of radius (each output pixel takes the most
// opaque source pixel in reach) and draws img over it.
func thickened(img *image.NRGBA, radius int) *image.RGBA {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	grown := image.NewNRGBA(image.Rect(0, 0, w+2*radius, h+2*radius))
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					from := img.PixOffset(x, y)
					to := grown.PixOffset(x+radius+dx, y+radius+dy)
					if img.Pix[from+3] > grown.Pix[to+3] {
						copy(grown.Pix[to:to+4], img.Pix[from:from+4])
					}
				}
			}
		}
	}
	out := newRGBA(w+2*radius, h+2*radius)
	draw.Draw(out, out.Rect, grown, image.Point{}, draw.Src)
	draw.Draw(out, image.Rect(radius, radius, radius+w, radius+h), img, image.Point{}, draw.Over)
	return out
}

func composeThumbnail(fieldSrc, judge, note *image.NRGBA, isolate bool) (*image.NRGBA, error) {
	layer := newRGBA(thumbSize, thumbSize)

	// Lane: the body below 24 % of the image, between the horizontal bounds
	// of its bottom rows, stretched to the trapezoid area; brightened,
	// saturated and made opaque over a dark floor.
	field := image.NewNRGBA(fieldSrc.Rect)
	copy(field.Pix, fieldSrc.Pix)
	if isolate {
		if err := isolateLaneBody(field); err != nil {
			return nil, err
		}
	}
	fh := field.Rect.Dy()
	// horizontal bounds of the bottom 8 rows with content (GarupaEditor's
	// collaboration lanes reach the image bottom; skin04 / skin13 end 21
	// rows above it)
	all, err := alphaBounds(field, 16, 0)
	if err != nil {
		return nil, fmt.Errorf("lane: %w", err)
	}
	fb, err := alphaBounds(field, 16, all.Max.Y-8)
	if err != nil {
		return nil, fmt.Errorf("lane: %w", err)
	}
	cutY := int(math.Round(float64(fh) * .24))
	laneH := thumbBottom - thumbTop
	lane := newRGBA(thumbWidth, laneH)
	scaleOver(lane, lane.Rect, field, image.Rect(fb.Min.X, cutY, fb.Max.X, fh))
	laneN := imgutil.ToNRGBA(lane)
	floor := [3]float64{30, 33, 42}
	for i := 0; i < len(laneN.Pix); i += 4 {
		a := float64(laneN.Pix[i+3]) / 255
		if a == 0 {
			continue
		}
		alpha := 1 - math.Pow(1-a, 3)
		var rgb [3]float64
		for c := 0; c < 3; c++ {
			rgb[c] = math.Min(255, float64(laneN.Pix[i+c])*1.65)
		}
		lum := rgb[0]*.299 + rgb[1]*.587 + rgb[2]*.114
		for c := 0; c < 3; c++ {
			v := math.Max(0, math.Min(255, lum+(rgb[c]-lum)*1.15))
			laneN.Pix[i+c] = uint8(math.Round(v*alpha + floor[c]*(1-alpha)))
		}
		laneN.Pix[i+3] = 255
	}
	fieldX := (thumbSize - thumbWidth) / 2
	draw.Draw(layer, image.Rect(fieldX, thumbTop, fieldX+thumbWidth, thumbBottom), laneN, image.Point{}, draw.Over)

	// Judge line: the middle 1160 px, scaled to the lane width, its most
	// covered rows (the line axis) on the lane's bottom edge, thickened.
	jw, jh := judge.Rect.Dx(), judge.Rect.Dy()
	crop := image.NewNRGBA(image.Rect(0, 0, 1160, jh))
	off := int(math.Round(float64(jw-1160) / 2))
	draw.Draw(crop, crop.Rect, judge, image.Pt(off, 0), draw.Src)
	peak, coverage := 0, make([]int, jh)
	for y := 0; y < jh; y++ {
		for x := 0; x < 1160; x++ {
			if crop.Pix[crop.PixOffset(x, y)+3] >= 128 {
				coverage[y]++
			}
		}
		peak = max(peak, coverage[y])
	}
	if peak == 0 {
		return nil, fmt.Errorf("judge line is empty")
	}
	sum, n := 0.0, 0
	for y, c := range coverage {
		if float64(c) >= float64(peak)*.95 {
			sum += float64(y) + .5
			n++
		}
	}
	axis := sum / float64(n)
	smallH := int(math.Round(float64(jh) * .1 * thumbS))
	small := newRGBA(thumbWidth-2*thumbS, smallH)
	scaleOver(small, small.Rect, crop, crop.Rect)
	judgeY := int(math.Round(thumbBottom - axis*float64(smallH)/float64(jh)))
	expanded := thickened(imgutil.ToNRGBA(small), thumbS)
	draw.Draw(layer, expanded.Rect.Add(image.Pt(fieldX, judgeY-thumbS)), expanded, image.Point{}, draw.Over)

	// Centre-lane note, fitted into 84 × 35 at y = 56.
	nb, err := alphaBounds(note, 8, 0)
	if err != nil {
		return nil, fmt.Errorf("note: %w", err)
	}
	factor := math.Min(84*thumbS/float64(nb.Dx()), 35*thumbS/float64(nb.Dy()))
	w, h := int(math.Round(float64(nb.Dx())*factor)), int(math.Round(float64(nb.Dy())*factor))
	x, y := int(math.Round(float64(thumbSize-w)/2)), int(math.Round(56*thumbS-float64(h)/2))
	scaleOver(layer, image.Rect(x, y, x+w, y+h), note, nb.Add(note.Rect.Min))

	// Dark outline: the combined alpha dilated by 3 px (at 1×).
	alpha := make([]uint8, thumbSize*thumbSize)
	for i := range alpha {
		alpha[i] = layer.Pix[i*4+3]
	}
	grown := dilate(alpha, thumbSize, thumbSize, 3*thumbS)
	outlined := newRGBA(thumbSize, thumbSize)
	border := image.NewNRGBA(outlined.Rect)
	for i, a := range grown {
		border.Pix[i*4], border.Pix[i*4+1], border.Pix[i*4+2], border.Pix[i*4+3] = 52, 52, 52, a
	}
	draw.Draw(outlined, outlined.Rect, border, image.Point{}, draw.Src)
	draw.Draw(outlined, outlined.Rect, layer, image.Point{}, draw.Over)

	result := newRGBA(128, 128)
	xdraw.CatmullRom.Scale(result, result.Rect, outlined, outlined.Rect, xdraw.Src, nil)
	return imgutil.ToNRGBA(result), nil
}
