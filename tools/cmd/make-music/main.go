// Command make-music writes assets/music.bin and assets/music-level-0N.bin.
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/music"
)

func main() {
	if err := music.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
