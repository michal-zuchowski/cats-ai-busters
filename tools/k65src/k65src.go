// Package k65src locates the repository and reads the game's K65 source as one
// text, in main.k65proj link order (the src/*.k65 files).
package k65src

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Root is the repository root (the directory holding main.k65proj).
func Root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// Path joins elements onto the repository root.
func Path(elem ...string) string {
	return filepath.Join(append([]string{Root()}, elem...)...)
}

// Files lists the source paths in main.k65proj (lines "<file>.k65 main"), in link order.
func Files() []string {
	data, err := os.ReadFile(Path("main.k65proj"))
	if err != nil {
		panic(err)
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[1] == "main" && strings.HasSuffix(f[0], ".k65") {
			out = append(out, Path(f[0]))
		}
	}
	return out
}

// Read returns the concatenated source joined with "\n"; exclude holds base
// file names such as "meow_sample.k65".
func Read(exclude ...string) string {
	skip := map[string]bool{}
	for _, e := range exclude {
		skip[e] = true
	}
	var parts []string
	for _, p := range Files() {
		if skip[filepath.Base(p)] {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			panic(err)
		}
		parts = append(parts, string(b))
	}
	return strings.Join(parts, "\n")
}
