// Command solve-levels proves levels 02-04 are completable in the headless sim.
//
// Usage: go run ./tools/cmd/solve-levels [level 1..3]   (needs a build and the ATR)
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/michal-zuchowski/cats-ai-busters/tools/solver"
)

func main() {
	levels := []int{1, 2, 3}
	if len(os.Args) > 1 {
		lv, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "usage: solve-levels [level 1..3]")
			os.Exit(2)
		}
		levels = []int{lv}
	}
	for _, lv := range levels {
		ok, n, paths, room, err := solver.Solve(lv, 15000)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		status := "solved"
		if !ok {
			status = fmt.Sprintf("STUCK in room %d", room)
		}
		fmt.Printf("level %02d: %s (%d tried)\n", lv+1, status, n)
		for _, p := range paths {
			fmt.Println("  ", strings.Join(p, " "))
		}
	}
}
