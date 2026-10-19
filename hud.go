package skin

import (
	"fmt"
	"image"
	"strings"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/unityasset"
)

const additiveNote = "additive art converted for normal alpha blending: alpha = max(rgb·a), rgb = rgb·a/alpha (matches the additive look over dark pixels)"

// catalogSource describes one texture of the render catalog (atlas rows of
// the APK atlases).
type catalogSource struct {
	file     string // catalog file name
	path     string // path under src/assets/game
	group    string
	prefix   string // name prefix for rows that would collide
	additive bool
	rename   map[string]string
}

var catalogSources = []catalogSource{
	{file: "combo-number.png", path: "atlas/bms/ui/iconcombonumber/combo-number.png", group: "hud"},
	{file: "rhythm-game-ui.png", path: "atlas/bms/ui/rhythmgameui/rhythm-game-ui.png", group: "hud"},
	{file: "rhythm-game-additive.png", path: "atlas/bms/ui/rhythmgameui/rhythm-game-additive.png", group: "hud", prefix: "additive/", additive: true},
	{file: "ui-additive-effect.png", path: "atlas/bms/ui/ui-additive-effect/ui-additive-effect.png", group: "hud", additive: true},
	{file: "tap-lane-effect-1.png", path: "atlas/bms/ui/tap-lane-effect/tap-lane-effect-1.png", group: "lane"},
	{file: "tap-lane-effect-2.png", path: "atlas/bms/ui/tap-lane-effect/tap-lane-effect-2.png", group: "lane"},
	{file: "tap-lane-effect-3.png", path: "atlas/bms/ui/tap-lane-effect/tap-lane-effect-3.png", group: "lane"},
	{file: "tap-lane-effect-4.png", path: "atlas/bms/ui/tap-lane-effect/tap-lane-effect-4.png", group: "lane"},
	{file: "sudden-line.png", path: "prefabs/bms/sudden/sudden-line.png", group: "lane"},
	{file: "high-rank-kira.png", path: "prefabs/bms/rhythmgamegauge/score/high-rank-kira.png", group: "hud",
		rename: map[string]string{"high-rank-kira": "ss_kira"}},
	{file: "high-rank-long-star.png", path: "prefabs/bms/rhythmgamegauge/score/high-rank-long-star.png", group: "hud",
		rename: map[string]string{"high-rank-long-star": "sss_star_long"}},
	{file: "high-rank-overlay.png", path: "prefabs/bms/rhythmgamegauge/score/high-rank-overlay.png", group: "hud",
		rename: map[string]string{"high-rank-overlay": "ss_overlay"}},
}

// Stand-alone textures without atlas rows (whole image = one sprite).
type plainSource struct {
	name, path, group, note string
	pivot                   [2]float64
	additive                bool
}

var plainSources = []plainSource{
	{name: "Tex_miniShine", path: "prefabs/bms/stage/stage-light.png", group: "stage", additive: true, pivot: [2]float64{0.5, 0.5},
		note: "stage light (light_color ≈180² and light_white ≈99.7² UI units)"},
	{name: "Tex_speaker", path: "prefabs/bms/stage/stage-speaker.png", group: "stage", pivot: [2]float64{0.5, 0.5}, note: "speaker, widget 188×256"},
	{name: "speaker_effect", path: "prefabs/bms/stage/stage-speaker-glow.png", group: "stage", additive: true, pivot: [2]float64{0.5, 0.5},
		note: "speaker glow, widget 118×114"},
	{name: "Pyllium", path: "prefabs/bms/stage/stage-psyllium.png", group: "stage", additive: true, pivot: [2]float64{0.5, 0},
		note: "psyllium 166×138: right half glow, left half core; widgets 64×128, pivot bottom"},
	{name: "startup_line_star", path: "prefabs/bms/information/startup-line-star.png", group: "startup", pivot: [2]float64{0.5, 0.5}},
	{name: "startup_title_base", path: "prefabs/bms/information/startup-title-base.png", group: "startup", pivot: [2]float64{0.5, 0.5}},
}

func addHUD(b *skinBuilder, o options) error {
	catData, err := apkFile("render-catalog.json")
	if err != nil {
		return err
	}
	cat, err := unityasset.ParseCatalog(catData)
	if err != nil {
		return err
	}
	for _, cs := range catalogSources {
		entry, err := cat.Entry(cs.file)
		if err != nil {
			return err
		}
		if entry.Profile == nil {
			return fmt.Errorf("catalog %s has no profile", cs.file)
		}
		tex, err := apkTexture(cs.path)
		if err != nil {
			return err
		}
		if cs.additive {
			tex = imgutil.AdditiveToAlpha(tex)
		}
		sprites, err := unityasset.CropRows(entry.Profile.AtlasRows, tex)
		if err != nil {
			return err
		}
		for _, key := range sortedKeys(sprites) {
			name := key
			if r, ok := cs.rename[key]; ok {
				name = r
			}
			note := ""
			if cs.additive {
				note = additiveNote
			}
			if err := b.addUnity(cs.prefix+name, cs.group, "APK "+cs.path, sprites[key], note); err != nil {
				return err
			}
		}
	}

	// UI common atlas: only level_mark (rank separator) is used in the live HUD.
	uiCommon, err := apkTexture("atlas/bms/ui/uicommon/ui-common.png")
	if err != nil {
		return err
	}
	mark, err := imgutil.Crop(uiCommon, 128, 4, 8, 5)
	if err != nil {
		return err
	}
	if err := b.add(SpriteMeta{Name: "level_mark", Kind: "sprite", Group: "hud", Source: "APK atlas/bms/ui/uicommon/ui-common.png (128,4,8,5)",
		Pivot: [2]float64{0.5, 0.5}, Note: "rank separator, widget 8×6"}, mark); err != nil {
		return err
	}

	for _, ps := range plainSources {
		if ps.group == "startup" && !o.startup {
			continue
		}
		img, err := apkTexture(ps.path)
		if err != nil {
			return err
		}
		note := ps.note
		if ps.additive {
			img = imgutil.AdditiveToAlpha(img)
			note = strings.TrimPrefix(note+"; "+additiveNote, "; ")
		}
		if err := b.add(SpriteMeta{Name: ps.name, Kind: "sprite", Group: ps.group, Source: "APK " + ps.path,
			Pivot: ps.pivot, Note: note}, img); err != nil {
			return err
		}
	}

	// Full Combo / All Perfect lettering (catalog group gameClear, one PNG each).
	for _, e := range cat.Groups["gameClear"] {
		if strings.HasPrefix(e.File, "Tex_parSet") || e.Profile == nil {
			continue // particle textures belong to the particle pack
		}
		tex, err := apkTexture("prefabs/bms/gameclear/" + e.File)
		if err != nil {
			return err
		}
		sprites, err := unityasset.CropRows(e.Profile.AtlasRows, tex)
		if err != nil {
			return err
		}
		for _, key := range sortedKeys(sprites) {
			if err := b.addUnity(stripHash(key), "clear", "APK prefabs/bms/gameclear/"+e.File, sprites[key], ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// addTints bakes the vertex colours the original applies at runtime, because
// Sonolus sprites cannot be tinted.
func addTints(b *skinBuilder) error {
	type tint struct {
		name string
		c    [4]float64
		note string
	}
	tints := []tint{
		{"hp_meter", [4]float64{110.0 / 255, 1, 105.0 / 255, 1}, "life fill, life ratio > 0.2 (#6EFF69)"},
		{"hp_meter", [4]float64{1, 0.6, 0.6, 1}, "life fill, life ratio ≤ 0.2"},
		{"additive/hp_meter", [4]float64{0.3655, 1, 0, 0.391}, "second life gauge (life > 1000), colour (0.3655,1,0,0.391)"},
		{"label_round_white", [4]float64{1, 0x3B / 255.0, 0x72 / 255.0, 1}, "Auto Live caption pill (#FF3B72)"},
		{"ss_overlay", [4]float64{1, 0.8667, 0.9294, 1}, "SS gauge Flash"},
		{"ss_kira", [4]float64{1, 0.9104, 0.9847, 0.8627}, "SS kira_1"},
		{"ss_kira", [4]float64{0.8443, 0.9838, 1, 1}, "SS kira_2"},
		{"ss_kira", [4]float64{0.9242, 1, 0.8255, 1}, "SS kira_3"},
		{"ss_kira", [4]float64{1, 1, 1, 0.8627}, "SS kira_4"},
		{"ss_kira", [4]float64{0.9923, 0.8915, 1, 1}, "SS kira_5"},
		{"ss_kira", [4]float64{0.9009, 0.9644, 1, 1}, "SS kira_6"},
		{"ss_kira", [4]float64{0.9439, 1, 0.9009, 0.8627}, "SS kira_7"},
		{"ss_kira", [4]float64{1, 0.9343, 0.8443, 1}, "SS kira_8"},
		{"Tex_miniShine", [4]float64{0, 55.0 / 255, 1, 1}, "stage light colour table: blue (0,55,255)"},
		{"Tex_miniShine", [4]float64{213.0 / 255, 1, 1, 1}, "stage light colour table: light blue (213,255,255)"},
		{"Pyllium", [4]float64{191.0 / 255, 238.0 / 255, 240.0 / 255, 1}, "psyllium default colour (191,238,240)"},
		{"Pyllium", [4]float64{200.0 / 255, 200.0 / 255, 200.0 / 255, 1}, "psyllium core colour (200,200,200)"},
	}
	for _, t := range tints {
		if _, err := b.addTint(t.name, t.c, t.note); err != nil {
			return err
		}
	}
	return nil
}

// convertAdditiveNoteSprites replaces the hold flash sprites (SpriteAdditive in
// the original) by their alpha-blending equivalent.
func convertAdditiveNoteSprites(b *skinBuilder) {
	for _, s := range b.sprites {
		n := s.meta.Name
		if strings.HasPrefix(n, "note_long_flash") || strings.HasPrefix(n, "habahiro/note_long_flash") {
			s.img = imgutil.AdditiveToAlpha(s.img)
			s.meta.Note = additiveNote + "; the original tints RGB 0.2→0.6→0.2, draw with that value as alpha"
		}
	}
}

var _ image.Image
