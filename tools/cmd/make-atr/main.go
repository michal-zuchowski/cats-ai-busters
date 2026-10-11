// Command make-atr builds out/cats-ai-busters.atr and assets/disk-chunks.bin.
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/atr"
)

func main() {
	if err := atr.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
