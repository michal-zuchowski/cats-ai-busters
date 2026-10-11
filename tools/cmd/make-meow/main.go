// Command make-meow writes assets/meow.pcm from assets/meow-source.wav.
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/meow"
)

func main() {
	if err := meow.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
