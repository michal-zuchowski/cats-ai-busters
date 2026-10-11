// Command make-level-art writes the level tile/room/camera assets.
// Usage: go run ./tools/cmd/make-level-art [preview-dir]
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/michal-zuchowski/cats-ai-busters/tools/catsprite"
	"github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
)

func main() {
	if err := levelart.WriteAssets(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		cat := catsprite.Drawing(0, 3)
		for n := 0; n < 3; n++ {
			if err := levelart.Preview(filepath.Join(os.Args[1], fmt.Sprintf("room%d.png", n)), n, &levelart.CatOverlay{X0: 40 + 30*n, Img: cat}); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}
