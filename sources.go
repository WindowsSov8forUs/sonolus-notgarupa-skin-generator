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

// stripHash removes the content-hash suffix of the game-clear file names
// ("AllPerfect_A-128DBB295661" → "AllPerfect_A").
func stripHash(name string) string {
	if i := strings.LastIndex(name, "-"); i > 0 && len(name)-i-1 == 12 {
		return name[:i]
	}
	return name
}

var _ = path.Base
