package catsprite

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// WriteAssets writes assets/cat-sprites.bin and assets/cat-frame-map.bin and
// returns the frames. A preview path named glass-knock-preview.png gets the
// glass-beat filmstrip, any other path the full filmstrip.
func WriteAssets(previewPath string) error {
	imgs := Frames()
	if len(imgs) != 58 {
		return fmt.Errorf("expected 58 frames, got %d", len(imgs))
	}
	data, mapping := PackedAtlas(imgs)
	if err := os.WriteFile(k65src.Path("assets", "cat-sprites.bin"), data, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(k65src.Path("assets", "cat-frame-map.bin"), mapping, 0o644); err != nil {
		return err
	}
	switch {
	case previewPath == "":
		return nil
	case filepath.Base(previewPath) == "glass-knock-preview.png":
		return GlassPreview(previewPath, imgs)
	}
	return Preview(previewPath, imgs, 10)
}
