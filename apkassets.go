package skin

// Assets built into the original game's APK (not available as downloadable
// asset bundles), shipped with the generator under assets/apk: the HUD,
// lane-effect and UI atlases with their atlas rows (render-catalog.json), the
// stage, Sudden and score-gauge art, the GameOverCover texture
// of the gameplay scene, and the sgm font.

import (
	"embed"
	"image"

	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/imgutil"
)

//go:embed assets/apk
var apkAssets embed.FS

// apkFile returns an APK asset by its path under assets/apk.
func apkFile(rel string) ([]byte, error) { return apkAssets.ReadFile("assets/apk/" + rel) }

// apkTexture decodes an APK texture.
func apkTexture(rel string) (*image.NRGBA, error) {
	data, err := apkFile(rel)
	if err != nil {
		return nil, err
	}
	return imgutil.Decode(data)
}
