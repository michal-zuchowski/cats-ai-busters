// Command make-intro-art writes the intro pictures and tables into assets/ (previews into out/).
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/introart"
)

func main() {
	if err := introart.WriteAssets(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
