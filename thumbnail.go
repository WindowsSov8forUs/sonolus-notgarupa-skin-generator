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

// thumbnail returns the PNG thumbnail of an item: key names the note skin
// and lane, fieldBundle is the lane's bundle path, note the centre-lane note
// and dir the directional skin's left body and arrow, right body and arrow. The base layer is composed once
// per (note skin, lane), the thumbnail once per item key.
func (ps *parts) thumbnail(key, fieldBundle string, note *image.NRGBA, dirKey string, dir [4]*image.NRGBA) ([]byte, error) {
	base, err := ps.once("thumbnail base "+key, func(e *partEntry) error {
		_, tex, err := bundleSprites(ps.bd, fieldBundle)
		if err != nil {
			return err
		}
		field, err := tex("bg_line_rhythm.png")
		if err != nil {
			return err
		}
		judge, err := tex("game_play_line.png")
		if err != nil {
			return err
		}
		layer, err := composeThumbnailBase(field, judge, note, fieldBundle == "ingameskin/fieldskin/skin_gbp2020")
		if err != nil {
			return fmt.Errorf("thumbnail %s: %w", key, err)
		}
		e.p = &part{base: layer}
		return nil
	})
	if err != nil {
		return nil, err
	}
	final, err := ps.once("thumbnail "+key+"|"+dirKey, func(e *partEntry) error {
		img, err := composeThumbnail(base.base, dir[0], dir[1], dir[2], dir[3])
		if err != nil {
			return fmt.Errorf("thumbnail %s|%s: %w", key, dirKey, err)
		}
		png, err := imgutil.EncodePNG(img)
		e.p = &part{thumbnail: png}
		return err
	})
	if err != nil {
		return nil, err
	}
	return final.thumbnail, nil
}

// once runs build once per key (concurrent callers wait) and returns its part.
func (ps *parts) once(key string, build func(e *partEntry) error) (*part, error) {
	ps.mu.Lock()
	e, ok := ps.cache[key]
	if !ok {
		e = &partEntry{}
		ps.cache[key] = e
	}
	ps.mu.Unlock()
	e.once.Do(func() { e.err = build(e) })
	return e.p, e.err
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

// Directional notes: the whole note of the centre lane, body
// (note_flick_<l|r>_3, 308×120 px) with its arrow (note_flick_top_<l|r>,
// 138×171 px) beside it at the rest offset of 1.6 note units (160 px at PPU
// 100), as the engine draws it.
const thumbArrowOffset = 160

// The thumbnail canvas (1× units) is larger than GarupaEditor's 128 so that
// both whole directional notes fit in a band above the lane: the left one
// in thumbDirBox, the right one mirrored; the base composition sits below.
const (
	thumbCanvas = 144
	thumbBaseDY = 14 // the base layer's offset below the band
	thumbOut    = 256
)

var thumbDirBox = [4]float64{3, 3, 69, 33}

// directionalNote draws a body with its arrow (left of the body when left).
func directionalNote(body, arrow *image.NRGBA, left bool) *image.RGBA {
	bw, bh := body.Rect.Dx(), body.Rect.Dy()
	aw, ah := arrow.Rect.Dx(), arrow.Rect.Dy()
	ax := thumbArrowOffset // arrow centre relative to the body centre
	if left {
		ax = -ax
	}
	minX := min(-bw/2, ax-aw/2)
	maxX := max(bw-bw/2, ax+aw-aw/2)
	h := max(bh, ah)
	out := newRGBA(maxX-minX, h)
	draw.Draw(out, image.Rect(-bw/2-minX, (h-bh)/2, -bw/2-minX+bw, (h-bh)/2+bh), body, body.Rect.Min, draw.Over)
	draw.Draw(out, image.Rect(ax-aw/2-minX, (h-ah)/2, ax-aw/2-minX+aw, (h-ah)/2+ah), arrow, arrow.Rect.Min, draw.Over)
	return out
}

// composeThumbnail is the item thumbnail: the base layer (lane, judge line,
// note) below the directional skin's whole left and right notes (Sonolus
// shows one image per item; GarupaEditor shows the directional skin
// separately), outlined, cropped to its content and scaled to a square.
func composeThumbnail(base *image.RGBA, bodyL, arrowL, bodyR, arrowR *image.NRGBA) (*image.NRGBA, error) {
	size := thumbCanvas * thumbS
	layer := newRGBA(size, size)
	dx, dy := (size-thumbSize)/2, thumbBaseDY*thumbS
	draw.Draw(layer, base.Rect.Add(image.Pt(dx, dy)), base, image.Point{}, draw.Over)
	for i, note := range []*image.RGBA{directionalNote(bodyL, arrowL, true), directionalNote(bodyR, arrowR, false)} {
		nb, err := alphaBounds(imgutil.ToNRGBA(note), 8, 0)
		if err != nil {
			return nil, fmt.Errorf("directional note: %w", err)
		}
		bw, bh := (thumbDirBox[2]-thumbDirBox[0])*thumbS, (thumbDirBox[3]-thumbDirBox[1])*thumbS
		factor := math.Min(bw/float64(nb.Dx()), bh/float64(nb.Dy()))
		w, h := int(math.Round(float64(nb.Dx())*factor)), int(math.Round(float64(nb.Dy())*factor))
		x := int(math.Round(thumbDirBox[0]*thumbS + (bw-float64(w))/2))
		y := int(math.Round(thumbDirBox[1]*thumbS + (bh-float64(h))/2))
		if i == 1 {
			x = size - x - w
		}
		scaleOver(layer, image.Rect(x, y, x+w, y+h), note, nb)
	}
	return outlineThumbnail(layer), nil
}

// composeThumbnailBase draws the lane, the judge line and the centre-lane
// note at 4× (GarupaEditor's composition).
func composeThumbnailBase(fieldSrc, judge, note *image.NRGBA, isolate bool) (*image.RGBA, error) {
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
	return layer, nil
}

// outlineThumbnail adds the dark outline and scales the 4× layer to 128×128.
func outlineThumbnail(layer *image.RGBA) *image.NRGBA {
	// Dark outline: the combined alpha dilated by 3 px (at 1×).
	size := layer.Rect.Dx()
	alpha := make([]uint8, size*size)
	for i := range alpha {
		alpha[i] = layer.Pix[i*4+3]
	}
	grown := dilate(alpha, size, size, 3*thumbS)
	outlined := newRGBA(size, size)
	border := image.NewNRGBA(outlined.Rect)
	for i, a := range grown {
		border.Pix[i*4], border.Pix[i*4+1], border.Pix[i*4+2], border.Pix[i*4+3] = 52, 52, 52, a
	}
	draw.Draw(outlined, outlined.Rect, border, image.Point{}, draw.Src)
	draw.Draw(outlined, outlined.Rect, layer, image.Point{}, draw.Over)

	// crop to the content, centred in a square with a 2-unit margin
	b, err := alphaBounds(imgutil.ToNRGBA(outlined), 1, 0)
	if err != nil {
		b = outlined.Rect
	}
	side := max(b.Dx(), b.Dy()) + 4*thumbS
	square := newRGBA(side, side)
	draw.Draw(square, b.Sub(b.Min).Add(image.Pt((side-b.Dx())/2, (side-b.Dy())/2)), outlined, b.Min, draw.Src)
	result := newRGBA(thumbOut, thumbOut)
	xdraw.CatmullRom.Scale(result, result.Rect, square, square.Rect, xdraw.Src, nil)
	return imgutil.ToNRGBA(result)
}
