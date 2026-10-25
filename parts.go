package skin

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/bestdori"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/unityasset"
)

// skillAdjustSprite is the field's skill overlay; the engine does not draw it.
const skillAdjustSprite = "game_play_line_skill_adjust_effect"

// defaultJudgePrefix names the default (skin00) judge_auto that every item
// carries for Auto Live: the original never applies a special judge skin
// there, and Auto Live only shows judge_auto.
const defaultJudgePrefix = "default/"

// part is the finished sprites of one source (a bundle, or the APK HUD set),
// shared read-only by every item that uses it.
type part struct {
	sprites   []*skinSprite
	thumbnail []byte      // a thumbnail entry's PNG (ps.thumbnail)
	base      *image.RGBA // a thumbnail base layer
}

// parts builds each part once per Generate call.
type parts struct {
	bd    *bestdori.Client
	mu    sync.Mutex
	cache map[string]*partEntry
}

type partEntry struct {
	once sync.Once
	p    *part
	err  error
}

func newParts(bd *bestdori.Client) *parts {
	return &parts{bd: bd, cache: map[string]*partEntry{}}
}

func (ps *parts) get(key string, build func(b *skinBuilder) error) (*part, error) {
	ps.mu.Lock()
	e, ok := ps.cache[key]
	if !ok {
		e = &partEntry{}
		ps.cache[key] = e
	}
	ps.mu.Unlock()
	e.once.Do(func() {
		b := newSkinBuilder()
		if e.err = build(b); e.err != nil {
			e.err = fmt.Errorf("%s: %w", key, e.err)
			return
		}
		convertAdditiveNoteSprites(b)
		b.trimTransparent()
		e.p = &part{sprites: b.sprites}
	})
	return e.p, e.err
}

// notes is a note skin: 45 sprites plus the long body textures.
func (ps *parts) notes(bundle string) (*part, error) {
	return ps.get("notes "+bundle, func(b *skinBuilder) error {
		noteBundle := "ingameskin/noteskin/" + bundle
		sprites, tex, err := bundleSprites(ps.bd, noteBundle)
		if err != nil {
			return err
		}
		for _, n := range sortedKeys(sprites) {
			if err := b.addUnity(n, "note", "bestdori "+noteBundle, sprites[n], ""); err != nil {
				return err
			}
		}
		for _, t := range []string{"longNoteLine", "longNoteLine2"} {
			img, err := tex(t + ".png")
			if err != nil {
				if t == "longNoteLine2" { // some skins only ship one body texture
					img, err = tex("longNoteLine.png")
				}
				if err != nil {
					return err
				}
			}
			note := "Long / straight Slide body"
			if t == "longNoteLine2" {
				note = "body of a Slide that has a hidden node"
			}
			if err := b.add(SpriteMeta{Name: t, Kind: "texture", Group: "note", Source: "bestdori " + noteBundle + "/" + t + ".png",
				Pivot: [2]float64{0.5, 0.5}, Note: note + "; stretched once over each segment, v=0 at the earlier node"}, img); err != nil {
				return err
			}
			// NoteMesh.Activate sets vertex RGB to 0.9 (alpha 0.8 from initMesh is left to the engine).
			if _, err := b.addTint(t, [4]float64{0.9, 0.9, 0.9, 1}, "vertex RGB 0.9 of an activated body baked in; draw with alpha 0.8"); err != nil {
				return err
			}
		}
		return nil
	})
}

// directional is a directional flick skin: note_flick_l/r_*, arrows and the
// back-line textures.
func (ps *parts) directional(bundle string) (*part, error) {
	return ps.get("directional "+bundle, func(b *skinBuilder) error {
		dirBundle := "ingameskin/noteskin/directionalflick" + bundle
		sprites, tex, err := bundleSprites(ps.bd, dirBundle)
		if err != nil {
			return err
		}
		for _, n := range sortedKeys(sprites) {
			if err := b.addUnity(n, "directional", "bestdori "+dirBundle, sprites[n], ""); err != nil {
				return err
			}
		}
		for _, t := range []string{"FlickNoteLine_l", "FlickNoteLine_r"} {
			img, err := tex(t + ".png")
			if err != nil {
				return err
			}
			if err := b.add(SpriteMeta{Name: t, Kind: "texture", Group: "directional", Source: "bestdori " + dirBundle + "/" + t + ".png",
				Pivot: [2]float64{0.5, 0.5}, Note: "multiple-directional back line (LineRenderer, stretch, width = note scale × 0.75)"}, img); err != nil {
				return err
			}
		}
		return nil
	})
}

// field is a lane skin: judge line (sprite, PPU 69) and lane image (UITexture).
func (ps *parts) field(bundle string) (*part, error) {
	return ps.get("field "+bundle, func(b *skinBuilder) error {
		fieldBundle := "ingameskin/fieldskin/" + bundle
		sprites, tex, err := bundleSprites(ps.bd, fieldBundle)
		if err != nil {
			return err
		}
		for _, n := range sortedKeys(sprites) {
			if n == skillAdjustSprite {
				continue
			}
			if err := b.addUnity(n, "field", "bestdori "+fieldBundle, sprites[n], ""); err != nil {
				return err
			}
		}
		lane, err := tex("bg_line_rhythm.png")
		if err != nil {
			return err
		}
		if err := b.add(SpriteMeta{Name: "bg_line_rhythm", Kind: "texture", Group: "field", Source: "bestdori " + fieldBundle + "/bg_line_rhythm.png",
			Pivot: [2]float64{0.5, 0}, Note: "UITexture 1160×610 UI units, pivot bottom, bottom edge on the judge line"}, lane); err != nil {
			return err
		}
		return addPreviewLane(b, previewLaneSprite, "field", "bestdori "+fieldBundle+"/bg_line_rhythm.png", lane)
	})
}

// judge is an NGUI Judge atlas (judge_perfect … judge_slow), names prefixed
// by prefix.
func (ps *parts) judge(bundle, prefix string) (*part, error) {
	return ps.get("judge "+prefix+bundle, func(b *skinBuilder) error {
		judgeBundle := "ingameskin/judgeskin/" + bundle
		files, err := ps.bd.Manifest(judgeBundle)
		if err != nil {
			return err
		}
		// Rips name the files <path>-judge.asset / -judge.png; a few ship
		// ".asset" / "Judge.png".
		var assetFile, pngFile string
		for _, f := range files {
			lower := strings.ToLower(f)
			switch {
			case strings.HasSuffix(lower, ".asset"):
				assetFile = f
			case strings.HasSuffix(lower, "judge.png"):
				pngFile = f
			}
		}
		if assetFile == "" || pngFile == "" {
			return fmt.Errorf("judge atlas files not found in %v", files)
		}
		aj, err := ps.bd.File(judgeBundle, assetFile)
		if err != nil {
			return err
		}
		pj, err := ps.bd.File(judgeBundle, pngFile)
		if err != nil {
			return err
		}
		tex, err := imgutil.Decode(pj)
		if err != nil {
			return err
		}
		sprites, err := unityasset.NGUIAtlas(aj, tex)
		if err != nil {
			return err
		}
		note := "NGUI sprite; widget size = sprite size (see contract)"
		if prefix == defaultJudgePrefix {
			note = "default judge_auto, used in Auto Live (the original applies a special judge skin only in manual play); " + note
			sprites = map[string]*unityasset.Sprite{"judge_auto": sprites["judge_auto"]}
			if sprites["judge_auto"] == nil {
				return fmt.Errorf("no judge_auto")
			}
		}
		for _, n := range sortedKeys(sprites) {
			if err := b.addUnity(prefix+n, "judge", "bestdori "+judgeBundle, sprites[n], note); err != nil {
				return err
			}
		}
		return nil
	})
}

// hud is the APK-built-in HUD, lane-tap, stage and clear art with the baked
// tints and the pre-rendered glyphs; the same for every item.
func (ps *parts) hud() (*part, error) {
	return ps.get("hud", func(b *skinBuilder) error {
		if err := addHUD(b); err != nil {
			return err
		}
		if err := addTints(b); err != nil {
			return fmt.Errorf("tints: %w", err)
		}
		if err := addGlyphs(b); err != nil {
			return fmt.Errorf("glyphs: %w", err)
		}
		return addPreviewLines(b)
	})
}

// habahiro is the HABAHIRO note atlas, field and lane-change flash.
func (ps *parts) habahiro() (*part, error) {
	return ps.get("habahiro", func(b *skinBuilder) error { return addHabahiro(b, ps.bd) })
}

// syncEdgeMarginMarker is present (1×1, transparent) in the items whose note
// skin has the master value noteSyncEdgeMargin 1.1 (TYPE7); without it the
// margin is 0. Sonolus engines cannot read item metadata, so the engine
// checks this sprite's existence.
const syncEdgeMarginMarker = "notgarupa/noteSyncEdgeMargin_1.1"

func (ps *parts) syncEdgeMargin() (*part, error) {
	return ps.get("noteSyncEdgeMargin", func(b *skinBuilder) error {
		return b.add(SpriteMeta{Name: syncEdgeMarginMarker, Kind: "marker", Group: "marker", Source: "master noteSyncEdgeMargin",
			Pivot: [2]float64{0.5, 0.5}, Note: "existence marker: the note skin's noteSyncEdgeMargin is 1.1 (simultaneous line ends pulled in by 1.1 × note scale)"},
			imgutil.Solid(1, 1, color.NRGBA{}))
	})
}
