// Command make-ai-scene writes assets/ai-portrait.pic and out/ai-cutscene-preview.png.
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/aiscene"
)

func main() {
	if err := aiscene.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
