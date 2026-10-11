// Command make-plat-levels writes assets/plat-rooms.bin and assets/plat-cry.pcm.
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/platlevels"
)

func main() {
	if err := platlevels.WriteAssets(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
