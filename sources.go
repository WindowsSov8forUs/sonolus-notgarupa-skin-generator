package skin

import (
	"fmt"
	"image"
	"path"
	"sort"
	"strings"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/bestdori"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/unityasset"
)

type options struct {
	noteSkin, directionalSkin, fieldSkin, judgeSkin string
	habahiro, startup, skillAdjust                  bool
}

// bundleSprites downloads a Bestdori bundle rip that has a .sprites dump and
// returns its named sprites plus a texture lookup for the bundle's PNG files.
func bundleSprites(bd *bestdori.Client, bundle string) (map[string]*unityasset.Sprite, func(string) (*image.NRGBA, error), error) {
	files, err := bd.Manifest(bundle)
	if err != nil {
		return nil, nil, err
	}
	cache := map[string]*image.NRGBA{}
	lookup := func(file string) (*image.NRGBA, error) {
		key := strings.ToLower(file)
		if img, ok := cache[key]; ok {
			return img, nil
		}
		for _, f := range files {
			if strings.ToLower(f) == key {
				data, err := bd.File(bundle, f)
				if err != nil {
					return nil, err
				}
				img, err := imgutil.Decode(data)
				if err != nil {
					return nil, fmt.Errorf("%s/%s: %w", bundle, f, err)
				}
				cache[key] = img
				return img, nil
			}
		}
		return nil, fmt.Errorf("bundle %s has no %s", bundle, file)
	}
	var spritesFile, bundleFile string
	for _, f := range files {
		switch {
		case f == ".sprites":
			spritesFile = f
		case strings.HasSuffix(f, ".bundle"):
			bundleFile = f
		}
	}
	if spritesFile == "" || bundleFile == "" {
		return map[string]*unityasset.Sprite{}, lookup, nil
	}
	sj, err := bd.File(bundle, spritesFile)
	if err != nil {
		return nil, nil, err
	}
	bj, err := bd.File(bundle, bundleFile)
	if err != nil {
		return nil, nil, err
	}
	sprites, err := unityasset.ExtractSprites(sj, bj, lookup)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", bundle, err)
	}
	return sprites, lookup, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// addNoteSkins adds the note, directional and field skins verbatim.
func addNoteSkins(b *skinBuilder, bd *bestdori.Client, o options) error {
	// Note skin: 45 sprites + long body textures.
	noteBundle := "ingameskin/noteskin/" + o.noteSkin
	sprites, tex, err := bundleSprites(bd, noteBundle)
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

	// Directional flick skin: note_flick_l/r_*, arrows, back-line textures.
	dirBundle := "ingameskin/noteskin/" + o.directionalSkin
	dsprites, dtex, err := bundleSprites(bd, dirBundle)
	if err != nil {
		return err
	}
	for _, n := range sortedKeys(dsprites) {
		if err := b.addUnity(n, "directional", "bestdori "+dirBundle, dsprites[n], ""); err != nil {
			return err
		}
	}
	for _, t := range []string{"FlickNoteLine_l", "FlickNoteLine_r"} {
		img, err := dtex(t + ".png")
		if err != nil {
			return err
		}
		if err := b.add(SpriteMeta{Name: t, Kind: "texture", Group: "directional", Source: "bestdori " + dirBundle + "/" + t + ".png",
			Pivot: [2]float64{0.5, 0.5}, Note: "multiple-directional back line (LineRenderer, stretch, width = note scale × 0.75)"}, img); err != nil {
			return err
		}
	}

	// Field skin: judge line (sprite, PPU 69) and lane image (UITexture).
	fieldBundle := "ingameskin/fieldskin/" + o.fieldSkin
	fsprites, ftex, err := bundleSprites(bd, fieldBundle)
	if err != nil {
		return err
	}
	for _, n := range sortedKeys(fsprites) {
		if n == "game_play_line_skill_adjust_effect" && !o.skillAdjust {
			continue // skill-only overlay; skills are not part of the port
		}
		if err := b.addUnity(n, "field", "bestdori "+fieldBundle, fsprites[n], ""); err != nil {
			return err
		}
	}
	lane, err := ftex("bg_line_rhythm.png")
	if err != nil {
		return err
	}
	if err := b.add(SpriteMeta{Name: "bg_line_rhythm", Kind: "texture", Group: "field", Source: "bestdori " + fieldBundle + "/bg_line_rhythm.png",
		Pivot: [2]float64{0.5, 0}, Note: "UITexture 1160×610 UI units, pivot bottom, bottom edge on the judge line"}, lane); err != nil {
		return err
	}

	return nil
}

// addJudgeSkin adds the NGUI Judge atlas (judge_perfect … judge_slow).
func addJudgeSkin(b *skinBuilder, bd *bestdori.Client, o options) error {
	bundle := "ingameskin/judgeskin/" + o.judgeSkin
	files, err := bd.Manifest(bundle)
	if err != nil {
		return err
	}
	var assetFile, pngFile string
	for _, f := range files {
		switch {
		case strings.HasSuffix(f, "-judge.asset"):
			assetFile = f
		case strings.HasSuffix(f, "-judge.png"):
			pngFile = f
		}
	}
	if assetFile == "" || pngFile == "" {
		return fmt.Errorf("%s: judge atlas files not found in %v", bundle, files)
	}
	aj, err := bd.File(bundle, assetFile)
	if err != nil {
		return err
	}
	pj, err := bd.File(bundle, pngFile)
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
	for _, n := range sortedKeys(sprites) {
		if err := b.addUnity(n, "judge", "bestdori "+bundle, sprites[n], "NGUI sprite; widget size = sprite size (see contract)"); err != nil {
			return err
		}
	}
	return nil
}

// stripHash removes the content-hash suffix of the game-clear file names
// ("AllPerfect_A-128DBB295661" → "AllPerfect_A").
func stripHash(name string) string {
	if i := strings.LastIndex(name, "-"); i > 0 && len(name)-i-1 == 12 {
		return name[:i]
	}
	return name
}

var _ = path.Base
