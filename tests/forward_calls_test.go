package tests

import (
	"regexp"
	"strings"
	"testing"
)

// K65 crashes on a bare call to a func declared later in link order; such
// calls must be written `call name` (see doc/tech/gameplay.md).
func TestNoBareForwardCalls(t *testing.T) {
	s := regexp.MustCompile(`//[^\n]*`).ReplaceAllString(src(), "")
	defAt := map[string]int{}
	for _, d := range regexp.MustCompile(`(?m)^\s*(?:func|naked)\s+([A-Za-z_]\w*)\s*\{`).FindAllStringSubmatchIndex(s, -1) {
		defAt[s[d[2]:d[3]]] = d[0]
	}
	for _, m := range regexp.MustCompile(`[A-Za-z_]\w*`).FindAllStringIndex(s, -1) {
		name := s[m[0]:m[1]]
		if at, ok := defAt[name]; !ok || m[0] >= at {
			continue
		}
		if m[0] > 0 && strings.ContainsAny(s[m[0]-1:m[0]], "<>&(+-=,#") {
			continue // address/operand use, not a call
		}
		if m[1] < len(s) && strings.ContainsAny(s[m[1]:m[1]+1], "=,+-)") {
			continue
		}
		before := strings.TrimRight(s[:m[0]], " \t")
		if strings.HasSuffix(before, "call") || strings.HasSuffix(before, "goto") {
			continue
		}
		t.Errorf("bare forward call to %s (joined source line %d); use `call %s`", name, strings.Count(s[:m[0]], "\n")+1, name)
	}
}
