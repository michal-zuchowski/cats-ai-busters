package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/platlevels"
)

// Portable L03 asset checks: generator output == committed asset, plus meaningful L03 content.
func TestL03Assets(t *testing.T) {
	nw := readAsset("plat-rooms.bin")
	if !bytes.Equal(nw, bytes.Join(platlevels.Rooms, nil)) || len(nw) != 10240 {
		t.Fatal("plat-rooms.bin differs from generator output")
	}
	ch := func(i int) []byte { return nw[i*1024 : (i+1)*1024] }
	M := func(i int) []byte { return ch(i)[960:] }
	A := func(i, r, c int) int { return int(ch(i)[r*40+c]) }
	all := func(cols func(yield func(int)), pred func(c int) bool) bool {
		ok := true
		cols(func(c int) { ok = ok && pred(c) })
		return ok
	}
	rng := func(a, b int) func(func(int)) {
		return func(y func(int)) {
			for c := a; c < b; c++ {
				y(c)
			}
		}
	}
	list := func(v ...int) func(func(int)) {
		return func(y func(int)) {
			for _, c := range v {
				y(c)
			}
		}
	}
	cat := func(fs ...func(func(int))) func(func(int)) {
		return func(y func(int)) {
			for _, f := range fs {
				f(y)
			}
		}
	}
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("check failed: %s", name)
		}
	}

	for i := 4; i < 8; i++ {
		exit := byte(168)
		if i == 5 || i == 7 {
			exit = 184
		}
		check("exit_y/vacuum row", M(i)[18] == exit && M(i)[8] == 21)
	}
	check("no head-height exit shelf", all(rng(35, 40), func(c int) bool { return A(5, 20, c)&127 != 70 }))
	check("exit floor", all(rng(35, 40), func(c int) bool { return A(5, 22, c) == 70 }))
	check("floor-height hatch", all(rng(19, 22), func(r int) bool { return A(5, r, 39) == 85 }))

	h := sha256.Sum256(append(append([]byte{}, nw[:7168]...), nw[8192:9216]...))
	check("regression hash", hex.EncodeToString(h[:]) == "7bfe2dc731e0b4090555a184032ecdec0e54a686aadab361354d88750b8e288c")
	check("R2 meta", M(5)[4] == 22 && M(5)[6] == 6 && M(5)[7] == 33 && M(5)[3] == 28 && M(5)[30] == 1)
	check("L04 meta", M(7)[30] == 2 && M(7)[6] == 12 && M(7)[7] == 12 && M(7)[3] == 19)
	check("pursuer only in R3", M(6)[29] != 0 && M(4)[29] == 0 && M(5)[29] == 0 && M(7)[29] == 0)
	check("R3 meta", M(6)[28] == 26 && M(4)[30] == 0 && M(6)[30] == 0)
	check("charger bay", all(list(22, 23, 24), func(c int) bool { return A(5, 21, c) == 12 && A(5, 20, c) == 88 }))
	check("no baked beam", all(rng(0, 40), func(c int) bool { return A(5, 19, c)&0x7F != 79 }))
	check("reader marks", all(list(12, 13, 14), func(c int) bool { return A(7, 21, c) == 12 }))
	check("reader tiles", all(list(12, 13, 14), func(c int) bool { return A(7, 20, c) == 83 }))
	check("R3 pit", all(rng(17, 33), func(c int) bool { return A(6, 22, c)&127 != 70 }))
	check("R3 floor", all(cat(rng(0, 17), rng(33, 40)), func(c int) bool { return A(6, 22, c) == 70 }))
	pit := true
	for r := 22; r < 24; r++ {
		for c := 23; c < 33; c++ {
			pit = pit && A(7, r, c) == 64
		}
	}
	check("R4 pit void", pit)
	check("R4 floor", all(cat(rng(0, 23), rng(33, 40)), func(c int) bool { return A(7, 22, c) == 70 }))
	check("R4 pit no floor", all(rng(23, 33), func(c int) bool { return A(7, 22, c) != 70 }))
	check("R4 patrol", M(7)[9] == 8 && M(7)[10] == 10 && M(7)[12] == 8)
	check("R4 lane", all(rng(23, 28), func(c int) bool { return A(7, 18, c) == 70 }))
	check("R4 robot floor", all(rng(8, 22), func(c int) bool { return A(7, 22, c) == 70 }))
	n := 0
	for r := 5; r < 18; r++ {
		for c := 0; c < 40; c++ {
			if A(7, r, c) != 64 {
				n++
			}
		}
	}
	check("R4 not empty", n >= 200)
	word := func(s string, c0 int) bool {
		for i := range s {
			if A(7, 18, c0+i) != int(s[i])-32 {
				return false
			}
		}
		return true
	}
	check("SCAN", word("SCAN", 11))
	check("EXIT", word("EXIT", 34))
}
