// Package skin builds the NotGarupa skin items: the original game's note,
// directional, lane and judge skins, the self-drawn HUD, lane-tap, stage and
// Full Combo / All Perfect art, and pre-rendered text. Sprite names are the
// original sprite names; see the engine's RESOURCE_CONTRACT.md.
//
// One item is built for every combination of the live settings that select
// skin art, as the original's master data defines them (internal/master): the
// normal note × lane × directional settings, the Collaboration / Limited rows
// with their fixed components, and the chart-forced HABAHIRO items.
//
// Generate streams the items in memory (for embedding in a server);
// CreateSourceTree writes them as a sonolus-pack source tree. The command in
// cmd/sonolus-notgarupa-skin-generator wraps both.
package skin

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/WindowsSov8forUs/sonolus-core-go/codec"
	"github.com/WindowsSov8forUs/sonolus-core-go/core/resource"
	"github.com/WindowsSov8forUs/sonolus-core-go/database"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/atlas"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/bestdori"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/master"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/packout"
)

// Source configures where downloadable assets come from.
type Source struct {
	// CacheDir caches downloads (Bestdori files under <CacheDir>/bestdori);
	// empty disables the disk cache.
	CacheDir string
	// Regions is the Bestdori server priority order (nil:
	// jp, en, tw, cn, kr). Each bundle comes from the first server that has
	// it, so server-exclusive assets are found too.
	Regions []string
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
	b.Regions = s.Regions
	if s.HTTPClient != nil {
		b.HTTP = s.HTTPClient
	}
	b.Offline, b.Log, b.Ctx = s.Offline, s.Log, ctx
	return b
}

// Options configures a run. The zero value builds every item.
type Options struct {
	Source Source
	// Workers is the number of items built at once (0: runtime.NumCPU()).
	Workers int
}

// Texture size limits: the ordinary items fit 2048×4096, the HABAHIRO items
// keep the full-resolution wide notes in 4096×8192.
const (
	maxTexture         = 4096
	maxTextureHabahiro = 8192
)

// Result is one generated skin item.
type Result struct {
	// Item is the database item; its resource fields (Thumbnail, Data,
	// Texture) are left empty for the caller to fill, e.g. with the server's
	// repository URLs of Files().
	Item database.DatabaseSkinItem
	// Habahiro marks the items of #HABAHIRO levels, which the server forces
	// for them as the original does.
	Habahiro  bool
	Data      resource.SkinData
	Texture   []byte // PNG
	Thumbnail []byte // PNG
	Contract  Contract
	// Axes are the item's live settings with their original names, for a
	// server's combination search (HABAHIRO items: none).
	Axes []Axis
}

// Axis is one live setting of an item: axis (notes, directionalFlick, lane,
// effect), bundle and display name.
type Axis = master.Axis

// Contract describes every sprite's geometry for the engine (written as
// contracts/<item>.json by the command).
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
	// NoteSyncEdgeMargin is the master value of the note skin.
	NoteSyncEdgeMargin float64 `json:"noteSyncEdgeMargin"`
}

// Files returns the item's resource files by item.json field name:
// "thumbnail", "data" (gzip-compressed SkinData) and "texture".
func (r *Result) Files() (map[string][]byte, error) {
	data, err := codec.Compress(r.Data)
	if err != nil {
		return nil, fmt.Errorf("compress skin data: %w", err)
	}
	return map[string][]byte{"thumbnail": r.Thumbnail, "data": data, "texture": r.Texture}, nil
}

// SourceTree is a sonolus-pack source tree that items are added to.
type SourceTree struct{ src *packout.Source }

// CreateSourceTree (re)creates root as a sonolus-pack source tree.
func CreateSourceTree(root string) (*SourceTree, error) {
	src, err := packout.Init(root, packout.T("NotGarupa Skins", "NotGarupa 皮肤", "NotGarupa 皮膚", "NotGarupa スキン"))
	if err != nil {
		return nil, err
	}
	return &SourceTree{src}, nil
}

// Add writes one item.
func (t *SourceTree) Add(r *Result) error {
	rel, err := t.src.ItemDir("skins", r.Item.Name, packout.Item{
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
	return t.src.WriteFiles(rel, files)
}

// Items returns the name of every item Generate builds, in generation order.
func Items() ([]string, error) {
	m, err := master.Load()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, s := range m.Skins() {
		names = append(names, s.Name())
	}
	return names, nil
}

// Generate builds every skin item and hands each to emit as soon as it is
// ready. emit is called from one goroutine at a time; items arrive in no
// particular order. The first error (from a build or from emit) stops the
// run and is returned.
func Generate(ctx context.Context, opts Options, emit func(*Result) error) error {
	m, err := master.Load()
	if err != nil {
		return err
	}
	specs := m.Skins()
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ps := newParts(opts.Source.bestdori(ctx))

	jobs := make(chan master.Skin)
	results := make(chan *Result)
	errs := make(chan error, workers+1)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for spec := range jobs {
				r, err := build(ps, spec)
				if err != nil {
					errs <- fmt.Errorf("%s: %w", spec.Name(), err)
					cancel()
					return
				}
				select {
				case results <- r:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, spec := range specs {
			select {
			case jobs <- spec:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()
	for r := range results {
		if err := emit(r); err != nil {
			cancel()
			for range results {
			}
			return err
		}
	}
	select {
	case err := <-errs:
		return err
	default:
	}
	return ctx.Err()
}

// build assembles one item from its parts, in a fixed order: note skin,
// directional skin, lane, judge, HUD, HABAHIRO additions, default judge and
// markers.
func build(ps *parts, spec master.Skin) (*Result, error) {
	notes, lane := spec.Notes, spec.Lane
	if spec.Habahiro {
		// The ordinary-named sprites of a HABAHIRO item: the lane shown before
		// the lane change, and skin00 notes (the forced habahiro note skin is
		// drawn through the habahiro/ sprites).
		notes, lane = "skin00", master.HabahiroPreField
	}
	var list []*part
	add := func(p *part, err error) error {
		if err == nil {
			list = append(list, p)
		}
		return err
	}
	if err := add(ps.notes(notes)); err != nil {
		return nil, err
	}
	if err := add(ps.directional(spec.Directional)); err != nil {
		return nil, err
	}
	if err := add(ps.field(lane)); err != nil {
		return nil, err
	}
	if err := add(ps.judge(spec.Judge, "")); err != nil {
		return nil, err
	}
	if err := add(ps.hud()); err != nil {
		return nil, err
	}
	if spec.Habahiro {
		if err := add(ps.habahiro()); err != nil {
			return nil, err
		}
	}
	if err := add(ps.judge(master.DefaultJudge, defaultJudgePrefix)); err != nil {
		return nil, err
	}
	if spec.NoteSyncEdgeMargin > 1 {
		if err := add(ps.syncEdgeMargin()); err != nil {
			return nil, err
		}
	}

	b := newSkinBuilder()
	for _, p := range list {
		for _, s := range p.sprites {
			if _, dup := b.byName[s.meta.Name]; dup {
				return nil, fmt.Errorf("duplicate sprite %q", s.meta.Name)
			}
			b.sprites = append(b.sprites, s)
			b.byName[s.meta.Name] = s
		}
	}

	limit := maxTexture
	if spec.Habahiro {
		limit = maxTextureHabahiro
	}
	items := make([]atlas.Item, 0, len(b.sprites))
	for _, s := range b.sprites {
		items = append(items, atlas.Item{Names: []string{s.meta.Name}, Image: s.img})
	}
	packed, err := atlas.PackPOT(items, limit, 2)
	if err != nil {
		return nil, err
	}
	data := resource.SkinData{
		Width:         float64(packed.Texture.Rect.Dx()),
		Height:        float64(packed.Texture.Rect.Dy()),
		Interpolation: true,
	}
	transform := resource.SkinDataTransform{}
	for k, v := range packout.IdentityTransform() {
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

	thumbName := "note_normal_3"
	if spec.Habahiro {
		thumbName = habahiroPrefix + "note_normal_2_3_4"
	}
	thumb, ok := b.byName[thumbName]
	if !ok {
		return nil, fmt.Errorf("no thumbnail sprite %s", thumbName)
	}
	thumbnail, err := imgutil.EncodePNG(thumb.img)
	if err != nil {
		return nil, err
	}

	title := spec.Title
	desc := fmt.Sprintf("Note: %s, directional: %s, lane: %s, judge: %s.", spec.Notes, spec.Directional, spec.Lane, spec.Judge)
	if spec.Habahiro {
		desc = fmt.Sprintf("For #HABAHIRO levels (the original forces the habahiro note skin, lane and judge skin): the skin00 lane until the lane change, the original HABAHIRO wide-note atlas, its long bodies, the lane-change field and flash; directional: %s.", spec.Directional)
	}
	groups := map[string]int{}
	for _, m := range metas {
		groups[m.Group]++
	}
	contract := Contract{
		Groups: groups,
		Options: ContractOptions{Directional: spec.Directional, Field: spec.Lane, Habahiro: spec.Habahiro, Judge: spec.Judge,
			Note: spec.Notes, NoteSyncEdgeMargin: spec.NoteSyncEdgeMargin},
		Skin:    spec.Name(),
		Sprites: metas,
		Texture: [2]int{packed.Texture.Rect.Dx(), packed.Texture.Rect.Dy()},
	}
	if spec.Habahiro {
		contract.Habahiro = habahiroContract()
	}
	return &Result{
		Item: database.DatabaseSkinItem{
			Name:        spec.Name(),
			Version:     packout.SkinVersion,
			Title:       packout.L(title),
			Subtitle:    packout.T("BanG Dream! Girls Band Party!", "BanG Dream! 少女乐团派对！", "BanG Dream! 少女樂團派對！", "バンドリ！ ガールズバンドパーティ！"),
			Author:      packout.T("GirlsBandParty", "GirlsBandParty", "GirlsBandParty", "GirlsBandParty"),
			Tags:        []database.DatabaseTag{},
			Description: packout.T(desc, desc, desc, desc),
		},
		Habahiro:  spec.Habahiro,
		Axes:      searchAxes(spec.Habahiro, spec.Axes),
		Data:      data,
		Texture:   texture,
		Thumbnail: thumbnail,
		Contract:  contract,
	}, nil
}

// searchAxes are an item's search axes (none for the hidden HABAHIRO items).
func searchAxes(habahiro bool, axes []Axis) []Axis {
	if habahiro {
		return nil
	}
	return axes
}
