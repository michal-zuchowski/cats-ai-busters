// Shared helpers for the ported tests (owned by the go-media-tools port).
//
//	src() string                     cached k65src.Read() (all src/*.k65 joined with "\n")
//	funcBody(src, name) string       python `re.search(r"(?:data|func|inline|naked) NAME \{(.*?)\n\}", re.S).group(1)`
//	                                 raw text, panics if missing (python asserts); superset of the
//	                                 func|data|naked variants used by most tests
//	funcBodyNorm(src, name) string   func-only regex body, comments stripped, whitespace collapsed (test_l03_routes)
//	funcBodyBraced(src, name) string `func NAME {` body found by brace matching, comments stripped,
//	                                 whitespace collapsed (test_level02 / test_level04)
//	varAddress(src, name) int        `var NAME = 0x....` as an int (panics if missing)
//	readAsset(name) []byte           bytes of assets/<name>
//	mustRead(path) []byte            os.ReadFile(path) or panic (path absolute or relative to repo root)
//	requireBuilt(t)                  t.Skip if out/cats-ai-busters.xex is missing
//	ensure(t, ok, format, args...)   non-fatal assertion (t.Errorf when !ok)
//	mustIndex(t, s, sub) int         strings.Index that fails the test if sub is absent (python str.index)
//	mustRIndex(t, s, sub) int        strings.LastIndex that fails the test if sub is absent (python str.rindex)
package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

var (
	srcOnce sync.Once
	srcText string
)

func src() string {
	srcOnce.Do(func() { srcText = k65src.Read() })
	return srcText
}

func funcBody(src, name string) string {
	re := regexp.MustCompile(`(?s)(?:data|func|inline|naked) ` + regexp.QuoteMeta(name) + ` \{(.*?)\n\}`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		panic("funcBody: no body for " + name)
	}
	return m[1]
}

var (
	lineComment = regexp.MustCompile(`//[^\n]*`)
	spaceRun    = regexp.MustCompile(`\s+`)
)

func normalize(s string) string {
	return strings.TrimSpace(spaceRun.ReplaceAllString(lineComment.ReplaceAllString(s, ""), " "))
}

func funcBodyNorm(src, name string) string {
	re := regexp.MustCompile(`(?s)func ` + regexp.QuoteMeta(name) + ` \{(.*?)\n\}`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		panic("funcBodyNorm: no body for " + name)
	}
	return normalize(m[1])
}

func funcBodyBraced(src, name string) string {
	head := "func " + name + " {"
	start := strings.Index(src, head)
	if start < 0 {
		panic("funcBodyBraced: no func " + name)
	}
	i := start + len(head) - 1
	depth := 0
	for j := i; j < len(src); j++ {
		switch src[j] {
		case '{':
			depth++
		case '}':
			depth--
		}
		if depth == 0 {
			return normalize(src[i+1 : j])
		}
	}
	panic("funcBodyBraced: unbalanced braces in " + name)
}

func varAddress(src, name string) int {
	re := regexp.MustCompile(`var ` + regexp.QuoteMeta(name) + ` = 0x([0-9A-Fa-f]+)`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		panic("varAddress: no var " + name)
	}
	v, err := strconv.ParseInt(m[1], 16, 64)
	if err != nil {
		panic(err)
	}
	return int(v)
}

func readAsset(name string) []byte { return mustRead(k65src.Path("assets", name)) }

func mustRead(path string) []byte {
	if !filepath.IsAbs(path) {
		path = k65src.Path(path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("mustRead: %v", err))
	}
	return b
}

func requireBuilt(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(k65src.Path("out", "cats-ai-busters.xex")); err != nil {
		t.Skip("out/cats-ai-busters.xex missing; build main.k65proj first")
	}
}

func ensure(t *testing.T, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Errorf(format, args...)
	}
}

func mustIndex(t *testing.T, s, sub string) int {
	t.Helper()
	i := strings.Index(s, sub)
	if i < 0 {
		t.Fatalf("%q not found", sub)
	}
	return i
}

func mustRIndex(t *testing.T, s, sub string) int {
	t.Helper()
	i := strings.LastIndex(s, sub)
	if i < 0 {
		t.Fatalf("%q not found", sub)
	}
	return i
}
