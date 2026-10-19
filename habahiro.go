package skin

import (
	"fmt"
	"strings"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/bestdori"
)

// HABAHIRO (multi-range / wide-note) mode. The original forces these bundles
// for every #HABAHIRO chart regardless of the user's skin settings:
//
//	ingameskin/noteskin/habahiro   wide-note atlas + its own long body textures
//	ingameskin/fieldskin/habahiro  lane image and judge line swapped in at the lane change
//	ingameskin/tapeffect/habahiro  Root_effect lane-change flash sprites (the
//	                               particles of that bundle belong to the particle pack)
//
// ingameskin/judgeskin/habahiro has the same pixels as skin00 and is not added.
// All sprites are named habahiro/<original name>; see HABAHIRO.md.

const (
	habahiroNoteBundle  = "ingameskin/noteskin/habahiro"
	habahiroFieldBundle = "ingameskin/fieldskin/habahiro"
	habahiroFlashBundle = "ingameskin/tapeffect/habahiro"
	habahiroPrefix      = "habahiro/"
)

// habahiroLaneChange is the clip time of the ChangeLane animation event of
// Root_effect (frame 25 at 60 Hz) at which the field sprites are swapped.
const habahiroLaneChange = 0.4166666567325592

var habahiroWidthPx = [8]int{0, 308, 524, 736, 952, 1164, 1380, 1596}

func addHabahiro(b *skinBuilder, bd *bestdori.Client) error {
	// Note atlas (RhythmGameSprites1..5,16): 6 families × 28 runs, slide-among
	// per width, up-flick icons per width, simultaneous_line.
	hs, htex, err := bundleSprites(bd, habahiroNoteBundle)
	if err != nil {
		return err
	}
	for _, n := range sortedKeys(hs) {
		if err := b.addUnity(habahiroPrefix+n, "habahiro", "bestdori "+habahiroNoteBundle, hs[n], habahiroNoteNote(n)); err != nil {
			return err
		}
	}
	for _, t := range []string{"longNoteLine", "longNoteLine2"} {
		img, err := htex(t + ".png")
		if err != nil {
			return err
		}
		note := "HABAHIRO Long / straight Slide body"
		if t == "longNoteLine2" {
			note = "HABAHIRO body of a Slide that has a hidden node"
		}
		name := habahiroPrefix + t
		if err := b.add(SpriteMeta{Name: name, Kind: "texture", Group: "habahiro", Source: "bestdori " + habahiroNoteBundle + "/" + t + ".png",
			Pivot: [2]float64{0.5, 0.5}, Note: note + "; stretched once over each segment, v=0 at the earlier node; mesh half-width × widthRate(w)"}, img); err != nil {
			return err
		}
		if _, err := b.addTint(name, [4]float64{0.9, 0.9, 0.9, 1}, "vertex RGB 0.9 of an activated body baked in; draw with alpha 0.8"); err != nil {
			return err
		}
	}

	// Field after the lane change. Same transforms as skin00; only the
	// pixels (and the judge-line height 38 → 43) change. The Sudden area
	// re-binds to the same bg_line_rhythm texture.
	fs, ftex, err := bundleSprites(bd, habahiroFieldBundle)
	if err != nil {
		return err
	}
	for _, n := range sortedKeys(fs) {
		if n == skillAdjustSprite {
			continue
		}
		note := fmt.Sprintf("replaces %s at laneChangeTime + %.10g s (ChangeLane event)", n, habahiroLaneChange)
		if err := b.addUnity(habahiroPrefix+n, "habahiro_field", "bestdori "+habahiroFieldBundle, fs[n], note); err != nil {
			return err
		}
	}
	lane, err := ftex("bg_line_rhythm.png")
	if err != nil {
		return err
	}
	if err := b.add(SpriteMeta{Name: habahiroPrefix + "bg_line_rhythm", Kind: "texture", Group: "habahiro_field",
		Source: "bestdori " + habahiroFieldBundle + "/bg_line_rhythm.png", Pivot: [2]float64{0.5, 0},
		Note: fmt.Sprintf("UITexture 1160×610 UI units, pivot bottom, bottom edge on the judge line; replaces bg_line_rhythm (lane image and Sudden area) at laneChangeTime + %.10g s", habahiroLaneChange)}, lane); err != nil {
		return err
	}

	// Lane-change flash (Root_effect SpriteRenderers, Sprites-Default = normal
	// alpha blending, so the pixels are used as they are).
	xs, _, err := bundleSprites(bd, habahiroFlashBundle)
	if err != nil {
		return err
	}
	for _, f := range habahiroFlash {
		s, ok := xs[f.name]
		if !ok {
			return fmt.Errorf("%s: no sprite %s", habahiroFlashBundle, f.name)
		}
		if err := b.addUnity(habahiroPrefix+f.name, "habahiro_flash", "bestdori "+habahiroFlashBundle, s, f.note); err != nil {
			return err
		}
	}
	return nil
}

var habahiroFlash = []struct{ name, note string }{
	{"bg_line_rhythm_flash", "lane-change flash, sorting 40 (mask VisibleOutsideMask); drawn 1160×610 UI units, x ±580, y −242..368 (RhythmGameLines-local, y up); alpha keys (0,0.001) (0.0833333,1) (0.75,1) (1,0.001), smoothstep between keys"},
	{"bg_line_rhythm_flash2", "lane-change flash, sorting 40 (mask VisibleOutsideMask); drawn 1160×610 UI units, x ±580, y −242..368; alpha keys (0,0.001) (0.3833333,1) (0.4166667,1) (0.45,1) (1,0.001), smoothstep between keys"},
	{"game_play_line_flash", "lane-change flash, sorting 41; drawn 1880×120 UI units, x ±940, y −302..−182 (centred on the judge line at −242); alpha keys (0,0.001) (0.0833333,1) (0.75,1) (1,0.001), smoothstep between keys"},
	{"game_play_line_flash2", "lane-change flash, sorting 41; drawn 1880×120 UI units, x ±940, y −302..−182; alpha keys (0,0.001) (0.3833333,1) (0.4166667,1) (0.45,1) (1,0.001), smoothstep between keys"},
}

// habahiroNoteNote documents the lane run a HABAHIRO note sprite stands for.
func habahiroNoteNote(name string) string {
	switch {
	case name == "simultaneous_line":
		return "HABAHIRO note skin simultaneous_line"
	case strings.HasPrefix(name, "note_slide_among"):
		w := 1
		fmt.Sscanf(strings.TrimPrefix(name, "note_slide_among_"), "%d", &w)
		return fmt.Sprintf("HABAHIRO Slide intermediate node, width %d (%d px)", w, habahiroWidthPx[w])
	case strings.HasPrefix(name, "note_flick_top"):
		return "HABAHIRO up-flick icon: note_flick_top (w=1), _2 (w=2), _3 (w≥3); local offset (0, 0.7)"
	}
	// note_<family>_<l0>_<l1>…: lanes l0..l0+w−1
	parts := strings.Split(name, "_")
	lanes := 0
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) == 1 && parts[i][0] >= '0' && parts[i][0] <= '6' {
			lanes++
			continue
		}
		break
	}
	if lanes == 0 || lanes > 7 {
		return "HABAHIRO (multi-range) note skin"
	}
	first := parts[len(parts)-lanes]
	last := parts[len(parts)-1]
	return fmt.Sprintf("HABAHIRO note, lanes %s..%s (width %d, %d px)", first, last, lanes, habahiroWidthPx[lanes])
}

// habahiroContract is the machine-readable summary of HABAHIRO.md.
func habahiroContract() map[string]any {
	return map[string]any{
		"note":         "sprite names and geometry the engine needs for #HABAHIRO levels; see HABAHIRO.md",
		"runName":      "habahiro/note_<family>_<start>_<start+1>_…_<start+w-1>, lanes 0..6, 1 ≤ w ≤ 7, start+w ≤ 7 (28 runs per family)",
		"runFamilies":  []string{"note_normal", "note_normal_16", "note_skill", "note_flick", "note_long", "note_long_flash"},
		"runIndex":     "idx(start,w) = Σ_{k=1}^{w−1}(8−k) + start = (w−1)(16−w)/2 + start (0..27)",
		"widthPx":      habahiroWidthPx[1:],
		"noteHeightPx": 120, "notePPU": 100, "notePivot": [2]float64{0.5, 0.5},
		"slideAmong":            "habahiro/note_slide_among (w=1), habahiro/note_slide_among_<w> (w=2..7)",
		"flickTop":              "habahiro/note_flick_top (w=1), _2 (w=2), _3 (w≥3)",
		"slideCentredRun":       "held Slide head body and Slide hold flash: size=w, start=floor((7−size)/2)+(size even && centreLane>3 ? 1 : 0)",
		"laneChangeSwapSeconds": habahiroLaneChange,
		"fieldSwap": map[string]string{
			"bg_line_rhythm":                     "habahiro/bg_line_rhythm",
			"game_play_line":                     "habahiro/game_play_line",
			"game_play_line_skill_adjust_effect": "habahiro/game_play_line_skill_adjust_effect",
			"longNoteLine":                       "habahiro/longNoteLine (whole level, not only after the lane change)",
			"longNoteLine2":                      "habahiro/longNoteLine2 (whole level)",
			"simultaneous_line":                  "habahiro/simultaneous_line (whole level)",
		},
		"flashDurationSeconds": 1.0,
		"flash": []map[string]any{
			{"sprite": "habahiro/bg_line_rhythm_flash", "rect": [4]float64{-580, -242, 580, 368}, "sorting": 40, "alphaKeys": [][2]float64{{0, 0.001}, {0.0833333, 1}, {0.75, 1}, {1, 0.001}}},
			{"sprite": "habahiro/bg_line_rhythm_flash2", "rect": [4]float64{-580, -242, 580, 368}, "sorting": 40, "alphaKeys": [][2]float64{{0, 0.001}, {0.3833333, 1}, {0.4166667, 1}, {0.45, 1}, {1, 0.001}}},
			{"sprite": "habahiro/game_play_line_flash", "rect": [4]float64{-940, -302, 940, -182}, "sorting": 41, "alphaKeys": [][2]float64{{0, 0.001}, {0.0833333, 1}, {0.75, 1}, {1, 0.001}}},
			{"sprite": "habahiro/game_play_line_flash2", "rect": [4]float64{-940, -302, 940, -182}, "sorting": 41, "alphaKeys": [][2]float64{{0, 0.001}, {0.3833333, 1}, {0.4166667, 1}, {0.45, 1}, {1, 0.001}}},
		},
		"flashRectSpace": "RhythmGameLines-local UI units, y up, (x0,y0,x1,y1); same mapping as the lane image (skin00 lane image bottom is at −240)",
		"flashAlpha":     "smoothstep between keys: v = a+(b−a)(3r²−2r³); RGB stays 1; normal alpha blend",
	}
}
