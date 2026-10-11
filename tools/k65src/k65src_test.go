package k65src

import (
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	src := Read()
	if !strings.Contains(src, "func check_detection") || !strings.Contains(src, "main {") {
		t.Fatal("project sources not concatenated")
	}
	if strings.Contains(Read("meow_sample.k65"), "data Meow") {
		t.Fatal("exclude ignored")
	}
}
