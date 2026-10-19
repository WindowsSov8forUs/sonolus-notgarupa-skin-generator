// Package skin builds the NotGarupa skin item: the original game's
// note/directional/field/judge skins, the self-drawn HUD, lane-tap, stage and
// Full Combo / All Perfect art, and pre-rendered text. Sprite names are the
// original sprite names; see the engine's RESOURCE_CONTRACT.md.
//
// Generate returns the skin item in memory (for embedding in a server);
// WriteSource writes it as a sonolus-pack source tree. The command in
// cmd/sonolus-notgarupa-skin-generator wraps both.
package skin

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/WindowsSov8forUs/sonolus-core-go/codec"
	"github.com/WindowsSov8forUs/sonolus-core-go/core/resource"
	"github.com/WindowsSov8forUs/sonolus-core-go/database"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/atlas"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/bestdori"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/packout"
)

// Source configures where downloadable assets come from.
type Source struct {
	// CacheDir caches downloads (Bestdori files under <CacheDir>/bestdori);
	// empty disables the disk cache.
	CacheDir string
	// Region is the Bestdori server region ("" = jp).
	Region string
	// Offline fails every download that is not already cached.
	Offline bool
	// HTTPClient is used for downloads (nil: a client with a 90 s timeout).
	HTTPClient *http.Client
	// Log receives one line per download (nil: silent).
	Log func(format string, args ...any)
}

func (s Source) bestdori(ctx context.Context) *bestdori.Client {
	cache := ""
	if s.CacheDir != "" {
		cache = filepath.Join(s.CacheDir, "bestdori")
	}
	b := bestdori.New(cache)
	if s.Region != "" {
		b.Region = s.Region
	}
	if s.HTTPClient != nil {
		b.HTTP = s.HTTPClient
	}
	b.Offline, b.Log, b.Ctx = s.Offline, s.Log, ctx
	return b
}

// Variant selects the skin item.
type Variant string

const (
	// Normal is the skin for ordinary levels.
	Normal Variant = "normal"
	// Habahiro is the separate item for #HABAHIRO levels: everything of the
	// normal skin plus the HABAHIRO note/field/flash sprites (habahiro/<name>;
	// see HABAHIRO.md).
	Habahiro Variant = "habahiro"
)

// Options selects what to build. The zero value builds the default item.
type Options struct {
	// Note is the note skin bundle ingameskin/noteskin/<id> ("" = skin00).
	Note string
	// Directional is the directional flick note skin bundle ("" =
	// directionalflickskin00).
	Directional string
	// Field is the field (lane) skin bundle ingameskin/fieldskin/<id> ("" =
	// skin00).
	Field string
	// Judge is the judge atlas bundle ingameskin/judgeskin/<id> ("" = skin00).
	Judge   string
	Variant Variant // "" = Normal
	// Startup includes the live startup direction art (startup_line_star,
	// startup_title_base).
	Startup bool
	// SkillAdjust includes game_play_line_skill_adjust_effect.
	SkillAdjust bool
	// MaxTexture is the maximum texture side, a power of two (0 = 4096, or
	// 8192 for Habahiro: full-resolution HABAHIRO notes need 4096×8192).
	MaxTexture int
	// Name is the skin item name ("" = notgarupa-<note>-<field>, plus
	// -habahiro for Habahiro).
	Name   string
	Source Source
}

// Result is one generated skin item.
type Result struct {
	// Item is the database item; its resource fields (Thumbnail, Data,
	// Texture) are left empty for the caller to fill, e.g. with the server's
	// repository URLs of Files().
	Item      database.DatabaseSkinItem
	Data      resource.SkinData
	Texture   []byte // PNG
	Thumbnail []byte // PNG; nil when the thumbnail sprite is missing
	Contract  Contract
}

// Contract describes every sprite's geometry for the engine (written as
// contract.json by the command).
type Contract struct {
	Groups   map[string]int  `json:"groups"`
	Habahiro map[string]any  `json:"habahiro,omitempty"`
	Options  ContractOptions `json:"options"`
	Skin     string          `json:"skin"`
	Sprites  []SpriteMeta    `json:"sprites"`
	Texture  [2]int          `json:"texture"`
}

// ContractOptions records the bundles the skin was built from.
type ContractOptions struct {
	Directional string `json:"directional"`
	Field       string `json:"field"`
	Habahiro    bool   `json:"habahiro"`
	Judge       string `json:"judge"`
	Note        string `json:"note"`
}

// Files returns the item's resource files by item.json field name:
// "thumbnail" (when present), "data" (gzip-compressed SkinData) and
// "texture".
func (r *Result) Files() (map[string][]byte, error) {
	data, err := codec.Compress(r.Data)
	if err != nil {
		return nil, fmt.Errorf("compress skin data: %w", err)
	}
	files := map[string][]byte{"data": data, "texture": r.Texture}
	if r.Thumbnail != nil {
		files["thumbnail"] = r.Thumbnail
	}
	return files, nil
}

// WriteSource (re)creates root as a sonolus-pack source tree holding the
// given skin items.
func WriteSource(root string, results ...*Result) error {
	src, err := packout.Init(root, packout.T("NotGarupa Skins", "NotGarupa 皮肤", "NotGarupa スキン"))
	if err != nil {
		return err
	}
	for _, r := range results {
		rel, err := src.ItemDir("skins", r.Item.Name, packout.Item{
			Version: r.Item.Version, Title: r.Item.Title, Subtitle: r.Item.Subtitle, Author: r.Item.Author,
			Tags: r.Item.Tags, Description: r.Item.Description,
		})
		if err != nil {
			return err
		}
		files, err := r.Files()
		if err != nil {
			return err
		}
		if err := src.WriteFiles(rel, files); err != nil {
			return err
		}
	}
	return nil
}

// Generate builds the skin item.
func Generate(ctx context.Context, opts Options) (*Result, error) {
	o := options{noteSkin: opts.Note, directionalSkin: opts.Directional, fieldSkin: opts.Field, judgeSkin: opts.Judge,
		startup: opts.Startup, skillAdjust: opts.SkillAdjust}
	for _, d := range []struct {
		v   *string
		def string
	}{{&o.noteSkin, "skin00"}, {&o.directionalSkin, "directionalflickskin00"}, {&o.fieldSkin, "skin00"}, {&o.judgeSkin, "skin00"}} {
		if *d.v == "" {
			*d.v = d.def
		}
	}
	switch opts.Variant {
	case "", Normal:
	case Habahiro:
		o.habahiro = true
	default:
		return nil, fmt.Errorf("unknown variant %q (normal, habahiro)", opts.Variant)
	}
	maxTexture := opts.MaxTexture
	if maxTexture <= 0 {
		maxTexture = 4096
		if o.habahiro {
			maxTexture = 8192
		}
	}
	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("notgarupa-%s-%s", o.noteSkin, o.fieldSkin)
		if o.habahiro {
			name += "-habahiro"
		}
	}

	bd := opts.Source.bestdori(ctx)
	b := newSkinBuilder()
	steps := []struct {
		what string
		run  func() error
	}{
		{"note skins", func() error { return addNoteSkins(b, bd, o) }},
		{"judge skin", func() error { return addJudgeSkin(b, bd, o) }},
		{"HUD / lane / stage / clear art", func() error { return addHUD(b, o) }},
		{"tints", func() error { return addTints(b) }},
		{"glyphs", func() error { return addGlyphs(b) }},
	}
	if o.habahiro {
		steps = append(steps, struct {
			what string
			run  func() error
		}{"HABAHIRO note / field / flash", func() error { return addHabahiro(b, bd, o) }})
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			return nil, fmt.Errorf("%s: %w", s.what, err)
		}
	}
	convertAdditiveNoteSprites(b)
	b.trimTransparent()

	items := make([]atlas.Item, 0, len(b.sprites))
	for _, s := range b.sprites {
		items = append(items, atlas.Item{Names: []string{s.meta.Name}, Image: s.img})
	}
	packed, err := atlas.PackPOT(items, maxTexture, 2)
	if err != nil {
		return nil, fmt.Errorf("%w (try a larger maximum texture size)", err)
	}
	data := resource.SkinData{
		Width:         float64(packed.Texture.Rect.Dx()),
		Height:        float64(packed.Texture.Rect.Dy()),
		Interpolation: true,
	}
	identity := packout.IdentityTransform()
	transform := resource.SkinDataTransform{}
	for k, v := range identity {
		transform[k] = resource.SkinDataExpression(v)
	}
	metas := b.sortedMeta()
	for _, m := range metas {
		p, _ := packed.Rect(m.Name)
		data.Sprites = append(data.Sprites, resource.SkinDataSprite{
			Name: resource.SkinSpriteName(m.Name), X: float64(p.X), Y: float64(p.Y), W: float64(p.W), H: float64(p.H),
			Transform: transform,
		})
	}
	texture, err := packed.EncodeTexture()
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("NotGarupa %s / %s", o.noteSkin, o.fieldSkin)
	if o.habahiro {
		title += " (HABAHIRO)"
	}
	desc := fmt.Sprintf("Note: %s, directional: %s, field: %s, judge: %s, HABAHIRO: %v.", o.noteSkin, o.directionalSkin, o.fieldSkin, o.judgeSkin, o.habahiro)
	if o.habahiro {
		desc += " For #HABAHIRO levels: everything of the normal skin plus the original HABAHIRO wide-note atlas, its long bodies, the lane-change field (habahiro/bg_line_rhythm, habahiro/game_play_line) and the lane-change flash."
	}
	thumbName := "note_normal_3"
	if o.habahiro {
		thumbName = habahiroPrefix + "note_normal_2_3_4"
	}
	var thumbnail []byte
	if thumb, ok := b.byName[thumbName]; ok {
		if thumbnail, err = imgutil.EncodePNG(thumb.img); err != nil {
			return nil, err
		}
	}
	groups := map[string]int{}
	for _, m := range metas {
		groups[m.Group]++
	}
	contract := Contract{
		Groups:  groups,
		Options: ContractOptions{Directional: o.directionalSkin, Field: o.fieldSkin, Habahiro: o.habahiro, Judge: o.judgeSkin, Note: o.noteSkin},
		Skin:    name,
		Sprites: metas,
		Texture: [2]int{packed.Texture.Rect.Dx(), packed.Texture.Rect.Dy()},
	}
	if o.habahiro {
		contract.Habahiro = habahiroContract()
	}
	return &Result{
		Item: database.DatabaseSkinItem{
			Name:        name,
			Version:     packout.SkinVersion,
			Title:       packout.T(title, title, title),
			Subtitle:    packout.T("BanG Dream! Girls Band Party! (original art)", "BanG Dream! 少女乐团派对！（原作素材）", "バンドリ！ ガールズバンドパーティ！（原作素材）"),
			Author:      packout.T("sonolus-notgarupa-skin-generator", "sonolus-notgarupa-skin-generator", "sonolus-notgarupa-skin-generator"),
			Tags:        []database.DatabaseTag{},
			Description: packout.T(desc, desc, desc),
		},
		Data:      data,
		Texture:   texture,
		Thumbnail: thumbnail,
		Contract:  contract,
	}, nil
}
