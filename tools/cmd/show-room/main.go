// Command show-room prints an ASCII preview of a platform room: show-room <room 0-9>.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/michal-zuchowski/cats-ai-busters/tools/platlevels"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: show-room <room 0-9>")
		os.Exit(2)
	}
	n, err := strconv.Atoi(os.Args[1])
	if err != nil || n < 0 || n >= len(platlevels.Rooms) {
		fmt.Fprintln(os.Stderr, "show-room: room must be 0-9")
		os.Exit(2)
	}
	fmt.Print(platlevels.Show(n))
}
