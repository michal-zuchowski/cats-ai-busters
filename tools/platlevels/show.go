package platlevels

import (
	"fmt"
	"strings"

	L "github.com/michal-zuchowski/cats-ai-busters/tools/levelart"
)

var showChars = func() map[int]byte {
	m := map[int]byte{L.Void: ' ', L.Floor: '=', L.Lift: 'L', L.Crumb: 'c', L.Ctrl: 'C', L.Wall: 'W',
		L.Hatch: 'H', L.Ceil: '#', L.CeilEdge: '#', L.CeilJoint: '#', L.Edge: '#', L.Conduit: '|', L.Panel: 'p'}
	for _, t := range append(L.FanA[:], L.FanB[:]...) {
		m[t] = 'f'
	}
	return m
}()

// Show renders room n as ASCII art (the output of tools/show_room.py).
func Show(n int) string {
	chunk := Rooms[n]
	var sb strings.Builder
	for r := 0; r < 24; r++ {
		fmt.Fprintf(&sb, "%2d ", r)
		for c := 0; c < 40; c++ {
			t := chunk[r*40+c]
			if ch, ok := showChars[int(t&0x7F)]; ok {
				sb.WriteByte(ch)
			} else if t != 0 {
				sb.WriteByte('r')
			} else {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("   ")
	for c := 0; c < 40; c++ {
		sb.WriteByte(byte('0' + c%10))
	}
	sb.WriteByte('\n')
	return sb.String()
}
