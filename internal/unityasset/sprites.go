// Package unityasset parses the JSON dumps that accompany Bestdori bundle rips
// (.sprites, .bundle, NGUI atlas .asset) and the atlas rows of the render
// catalog, and crops the named sprites out of their atlas textures.
package unityasset

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"path"
	"strings"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
)

// Sprite is one named sprite with the geometry the engine needs.
type Sprite struct {
	Name  string
	Image *image.NRGBA
	// PivotX/PivotY are normalised with a bottom-left origin (Unity convention).
	PivotX, PivotY float64
	// PixelsPerUnit is 0 for NGUI sprites (they are sized in UI units = pixels).
	PixelsPerUnit float64
	// Border is the nine-slice border in pixels (left, right, top, bottom).
	BorderL, BorderR, BorderT, BorderB int
}

type rawSprite struct {
	Base struct {
		Name string `json:"m_Name"`
		Rect struct {
			X, Y, Width, Height float64
		} `json:"m_Rect"`
		Border struct {
			X, Y, Z, W float64
		} `json:"m_Border"`
		PixelsToUnits float64                `json:"m_PixelsToUnits"`
		Pivot         struct{ X, Y float64 } `json:"m_Pivot"`
		RD            struct {
			Texture struct {
				PathID json.RawMessage `json:"m_PathID"`
			} `json:"texture"`
		} `json:"m_RD"`
	} `json:"Base"`
}

type rawBundle struct {
	Base struct {
		PreloadTable []struct {
			PathID json.RawMessage `json:"m_PathID"`
		} `json:"m_PreloadTable"`
		Container map[string]struct {
			PreloadIndex int `json:"preloadIndex"`
			PreloadSize  int `json:"preloadSize"`
		} `json:"m_Container"`
	} `json:"Base"`
}

func rawID(m json.RawMessage) string {
	s := strings.TrimSpace(string(m))
	return strings.Trim(s, `"`)
}

// TextureLookup returns a decoded atlas texture by (lower-case) file name.
type TextureLookup func(file string) (*image.NRGBA, error)

// ExtractSprites crops every sprite of a Bestdori `.sprites` dump. The bundle
// dump maps each sprite's texture path id to its PNG file.
func ExtractSprites(spritesJSON, bundleJSON []byte, textures TextureLookup) (map[string]*Sprite, error) {
	var sprites []rawSprite
	if err := json.Unmarshal(spritesJSON, &sprites); err != nil {
		return nil, fmt.Errorf(".sprites: %w", err)
	}
	var bundle rawBundle
	if err := json.Unmarshal(bundleJSON, &bundle); err != nil {
		return nil, fmt.Errorf(".bundle: %w", err)
	}
	textureFile := map[string]string{}
	for assetPath, entry := range bundle.Base.Container {
		file := strings.ToLower(path.Base(strings.ReplaceAll(assetPath, `\`, "/")))
		if !strings.HasSuffix(file, ".png") {
			continue
		}
		for i := entry.PreloadIndex; i < entry.PreloadIndex+entry.PreloadSize && i < len(bundle.Base.PreloadTable); i++ {
			id := rawID(bundle.Base.PreloadTable[i].PathID)
			if _, ok := textureFile[id]; !ok {
				textureFile[id] = file
			}
		}
	}
	out := map[string]*Sprite{}
	for _, s := range sprites {
		b := s.Base
		if b.Name == "" {
			continue
		}
		file, ok := textureFile[rawID(b.RD.Texture.PathID)]
		if !ok {
			return nil, fmt.Errorf("sprite %s: texture path id not found in bundle", b.Name)
		}
		tex, err := textures(file)
		if err != nil {
			return nil, fmt.Errorf("sprite %s: %w", b.Name, err)
		}
		img, err := imgutil.CropBottomLeft(tex, iround(b.Rect.X), iround(b.Rect.Y), iround(b.Rect.Width), iround(b.Rect.Height))
		if err != nil {
			return nil, fmt.Errorf("sprite %s: %w", b.Name, err)
		}
		// Unity m_Border is (left, bottom, right, top).
		out[b.Name] = &Sprite{
			Name: b.Name, Image: img,
			PivotX: b.Pivot.X, PivotY: b.Pivot.Y,
			PixelsPerUnit: b.PixelsToUnits,
			BorderL:       iround(b.Border.X), BorderB: iround(b.Border.Y),
			BorderR: iround(b.Border.Z), BorderT: iround(b.Border.W),
		}
	}
	return out, nil
}

// NGUIAtlas parses an NGUI UIAtlas MonoBehaviour dump (`mSprites`), whose
// rects use a top-left origin.
func NGUIAtlas(assetJSON []byte, texture *image.NRGBA) (map[string]*Sprite, error) {
	var raw struct {
		Base struct {
			Sprites []struct {
				Name                                                 string
				X, Y, Width, Height                                  int
				BorderLeft, BorderRight, BorderTop, BorderBottom     int
				PaddingLeft, PaddingRight, PaddingTop, PaddingBottom int
			} `json:"mSprites"`
		} `json:"Base"`
	}
	if err := json.Unmarshal(assetJSON, &raw); err != nil {
		return nil, fmt.Errorf("NGUI atlas: %w", err)
	}
	out := map[string]*Sprite{}
	for _, s := range raw.Base.Sprites {
		img, err := imgutil.Crop(texture, s.X, s.Y, s.Width, s.Height)
		if err != nil {
			return nil, fmt.Errorf("NGUI sprite %s: %w", s.Name, err)
		}
		out[s.Name] = &Sprite{
			Name: s.Name, Image: img, PivotX: 0.5, PivotY: 0.5,
			BorderL: s.BorderLeft, BorderR: s.BorderRight, BorderT: s.BorderTop, BorderB: s.BorderBottom,
		}
	}
	return out, nil
}

// CatalogRow is one atlas row of the render catalog (APK atlases)
// (top-left origin rects).
type CatalogRow struct {
	ExactKey      string  `json:"exactKey"`
	X             int     `json:"x"`
	Y             int     `json:"y"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	PivotX        float64 `json:"pivotX"`
	PivotY        float64 `json:"pivotY"`
	PixelsPerUnit float64 `json:"pixelsPerUnit"`
	BorderLeft    int     `json:"borderLeft"`
	BorderRight   int     `json:"borderRight"`
	BorderTop     int     `json:"borderTop"`
	BorderBottom  int     `json:"borderBottom"`
}

type CatalogEntry struct {
	File    string `json:"file"`
	Profile *struct {
		LogicalAssetID  string `json:"logicalAssetId"`
		Role            string `json:"role"`
		Width           int    `json:"width"`
		Height          int    `json:"height"`
		TextureSettings *struct {
			BlendMode string `json:"blendMode"`
		} `json:"textureSettings"`
		AtlasRows []CatalogRow `json:"atlasRows"`
	} `json:"profile"`
}

type Catalog struct {
	Groups map[string][]CatalogEntry `json:"groups"`
}

func ParseCatalog(data []byte) (*Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("render catalog: %w", err)
	}
	return &c, nil
}

// Entry finds a catalog entry by file name in any group.
func (c *Catalog) Entry(file string) (*CatalogEntry, error) {
	for _, g := range c.Groups {
		for i := range g {
			if g[i].File == file {
				return &g[i], nil
			}
		}
	}
	return nil, fmt.Errorf("render catalog has no entry for %s", file)
}

// CropRows crops every atlas row of a catalog entry from its texture.
func CropRows(rows []CatalogRow, texture *image.NRGBA) (map[string]*Sprite, error) {
	out := map[string]*Sprite{}
	for _, r := range rows {
		img, err := imgutil.Crop(texture, r.X, r.Y, r.Width, r.Height)
		if err != nil {
			return nil, fmt.Errorf("catalog row %s: %w", r.ExactKey, err)
		}
		// Catalog pivots use a top-left origin; Sprite uses Unity's bottom-left.
		out[r.ExactKey] = &Sprite{
			Name: r.ExactKey, Image: img, PivotX: r.PivotX, PivotY: 1 - r.PivotY, PixelsPerUnit: r.PixelsPerUnit,
			BorderL: r.BorderLeft, BorderR: r.BorderRight, BorderT: r.BorderTop, BorderB: r.BorderBottom,
		}
	}
	return out, nil
}

func iround(v float64) int { return int(math.Round(v)) }
