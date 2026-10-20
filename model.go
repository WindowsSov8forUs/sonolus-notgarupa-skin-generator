package skin

import (
	"fmt"
	"image"
	"sort"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/unityasset"
)

// SpriteMeta is what the contract records for every skin sprite so that the
// engine can compute exact geometry without guessing.
type SpriteMeta struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"` // sprite | texture | slice | tint | glyph | text
	Group  string `json:"group"`
	Source string `json:"source"`
	W      int    `json:"w"` // source pixels (glyphs: logical font pixels)
	H      int    `json:"h"`
	// Pivot is normalised with a bottom-left origin (Unity convention).
	Pivot         [2]float64 `json:"pivot"`
	PixelsPerUnit float64    `json:"ppu,omitempty"`
	// Border is the nine-slice border (left, right, top, bottom) in pixels.
	Border *[4]int `json:"border,omitempty"`
	// Slice parts: parent sprite and the part rect inside it (top-left origin).
	Of   string  `json:"of,omitempty"`
	Part *[4]int `json:"part,omitempty"`
	// Tint baked into the pixels (RGBA 0..1) for sprites the original tints.
	Tint *[4]float64 `json:"tint,omitempty"`
	// Glyph metrics in logical font pixels.
	Glyph *GlyphMeta `json:"glyph,omitempty"`
	// Supersampling factor of the stored image relative to W×H.
	Scale float64 `json:"scale,omitempty"`
	// Trim: fully transparent margins (left, top, right, bottom) removed from
	// the original rect of Full size; W/H and Pivot describe the trimmed image,
	// so drawing W×H/PPU around the pivot reproduces the original placement.
	Trim *[4]int `json:"trim,omitempty"`
	Full *[2]int `json:"full,omitempty"`
	Note string  `json:"note,omitempty"`
}

// GlyphMeta are the metrics of a pre-rendered glyph or text sprite.
type GlyphMeta struct {
	Font     string  `json:"font"`
	Size     float64 `json:"size"`
	Advance  float64 `json:"advance"`
	Ascent   float64 `json:"ascent"`
	Descent  float64 `json:"descent"`
	OriginX  float64 `json:"originX"` // pen origin x inside the image (logical px)
	Text     string  `json:"text"`
	SpacingX float64 `json:"spacingX,omitempty"`
}

type skinSprite struct {
	meta SpriteMeta
	img  *image.NRGBA
}

type skinBuilder struct {
	sprites []*skinSprite
	byName  map[string]*skinSprite
}

func newSkinBuilder() *skinBuilder { return &skinBuilder{byName: map[string]*skinSprite{}} }

func (b *skinBuilder) add(meta SpriteMeta, img *image.NRGBA) error {
	if _, dup := b.byName[meta.Name]; dup {
		return fmt.Errorf("duplicate skin sprite name %q", meta.Name)
	}
	if meta.W == 0 {
		meta.W, meta.H = img.Rect.Dx(), img.Rect.Dy()
	}
	s := &skinSprite{meta: meta, img: img}
	b.sprites = append(b.sprites, s)
	b.byName[meta.Name] = s
	return nil
}

// addUnity adds a sprite parsed from a Unity/NGUI/catalog source, plus its
// slice parts when it has a nine-slice border.
func (b *skinBuilder) addUnity(name, group, source string, s *unityasset.Sprite, note string) error {
	meta := SpriteMeta{
		Name: name, Kind: "sprite", Group: group, Source: source,
		Pivot: [2]float64{s.PivotX, s.PivotY}, PixelsPerUnit: s.PixelsPerUnit, Note: note,
	}
	if s.BorderL|s.BorderR|s.BorderT|s.BorderB != 0 {
		meta.Border = &[4]int{s.BorderL, s.BorderR, s.BorderT, s.BorderB}
	}
	if err := b.add(meta, s.Image); err != nil {
		return err
	}
	if meta.Border != nil {
		return b.addSlices(name, group, s.Image, *meta.Border, nil)
	}
	return nil
}

// addSlices cuts a sliced sprite into #L/#M/#R (horizontal border only) or
// #TL..#BR (any vertical border) parts. Zero-sized parts are skipped.
func (b *skinBuilder) addSlices(name, group string, img *image.NRGBA, border [4]int, tint *[4]float64) error {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	l, r, t, bt := border[0], border[1], border[2], border[3]
	xs := [][3]any{{"L", 0, l}, {"M", l, w - l - r}, {"R", w - r, r}}
	if t == 0 && bt == 0 {
		for _, x := range xs {
			if err := b.addPart(name, group, img, x[0].(string), x[1].(int), 0, x[2].(int), h, tint); err != nil {
				return err
			}
		}
		return nil
	}
	ys := [][3]any{{"T", 0, t}, {"", t, h - t - bt}, {"B", h - bt, bt}}
	for _, y := range ys {
		for _, x := range xs {
			suffix := y[0].(string) + x[0].(string)
			switch suffix {
			case "TM":
				suffix = "T"
			case "BM":
				suffix = "B"
			}
			if err := b.addPart(name, group, img, suffix, x[1].(int), y[1].(int), x[2].(int), y[2].(int), tint); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *skinBuilder) addPart(name, group string, img *image.NRGBA, suffix string, x, y, w, h int, tint *[4]float64) error {
	if w <= 0 || h <= 0 {
		return nil
	}
	part, err := imgutil.Crop(img, x, y, w, h)
	if err != nil {
		return fmt.Errorf("slice %s#%s: %w", name, suffix, err)
	}
	return b.add(SpriteMeta{
		Name: name + "#" + suffix, Kind: "slice", Group: group, Of: name,
		Part: &[4]int{x, y, w, h}, Pivot: [2]float64{0.5, 0.5}, Tint: tint,
	}, part)
}

// addTint adds `<name>@<RRGGBB[AA]>`, a tinted copy of an existing sprite,
// including tinted slice parts when the source is sliced.
func (b *skinBuilder) addTint(name string, c [4]float64, note string) (string, error) {
	src, ok := b.byName[name]
	if !ok {
		return "", fmt.Errorf("tint: no sprite %q", name)
	}
	tinted := imgutil.Tint(src.img, c[0], c[1], c[2], c[3])
	tname := name + "@" + hexColor(c)
	meta := src.meta
	meta.Name, meta.Kind, meta.Tint, meta.Note = tname, "tint", &c, note
	meta.Source = "tint of " + name
	if err := b.add(meta, tinted); err != nil {
		return "", err
	}
	if meta.Border != nil {
		if err := b.addSlices(tname, meta.Group, tinted, *meta.Border, &c); err != nil {
			return "", err
		}
	}
	return tname, nil
}

func hexColor(c [4]float64) string {
	to := func(v float64) int { return int(v*255 + 0.5) }
	s := fmt.Sprintf("%02X%02X%02X", to(c[0]), to(c[1]), to(c[2]))
	if c[3] < 1 {
		s += fmt.Sprintf("%02X", to(c[3]))
	}
	return s
}

func (b *skinBuilder) sortedMeta() []SpriteMeta {
	out := make([]SpriteMeta, 0, len(b.sprites))
	for _, s := range b.sprites {
		out = append(out, s.meta)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// trimTransparent removes fully transparent margins of plain sprites and
// textures (not slices, glyphs or sliced sprites) and fixes up the pivot.
func (b *skinBuilder) trimTransparent() {
	for _, s := range b.sprites {
		// Only free-standing stage art is trimmed; note, judge, HUD and lane
		// sprites keep their original rects so the engine can use uniform
		// per-family geometry.
		if s.meta.Group != "stage" {
			continue
		}
		if (s.meta.Kind != "sprite" && s.meta.Kind != "tint") || s.meta.Border != nil || s.meta.Scale != 0 {
			continue
		}
		img := s.img
		W, H := img.Rect.Dx(), img.Rect.Dy()
		minX, minY, maxX, maxY := W, H, -1, -1
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				if img.Pix[img.PixOffset(x, y)+3] != 0 {
					minX, minY = min(minX, x), min(minY, y)
					maxX, maxY = max(maxX, x), max(maxY, y)
				}
			}
		}
		if maxX < 0 {
			minX, minY, maxX, maxY = 0, 0, 0, 0 // fully transparent: keep 1 px
		}
		l, t, r, bt := minX, minY, W-1-maxX, H-1-maxY
		if l+t+r+bt == 0 {
			continue
		}
		w, h := maxX-minX+1, maxY-minY+1
		cropped, _ := imgutil.Crop(img, minX, minY, w, h)
		s.img = cropped
		px := (s.meta.Pivot[0]*float64(W) - float64(l)) / float64(w)
		py := (s.meta.Pivot[1]*float64(H) - float64(bt)) / float64(h)
		s.meta.Trim = &[4]int{l, t, r, bt}
		s.meta.Full = &[2]int{W, H}
		s.meta.W, s.meta.H = w, h
		s.meta.Pivot = [2]float64{px, py}
	}
}
