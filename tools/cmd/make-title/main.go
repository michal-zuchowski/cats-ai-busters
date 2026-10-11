// Command make-title writes assets/title-screen.pic from assets/title-screen.png.
package main

import (
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/title"
)

func main() {
	if err := title.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
