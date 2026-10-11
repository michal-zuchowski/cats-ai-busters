// Command make-cat-sprite writes assets/cat-sprites.bin and assets/cat-frame-map.bin.
// Usage: go run ./tools/cmd/make-cat-sprite [preview.png]
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/catsprite"
)

func main() {
	preview := ""
	if len(os.Args) > 1 {
		preview = os.Args[1]
	}
	if err := catsprite.WriteAssets(preview); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
