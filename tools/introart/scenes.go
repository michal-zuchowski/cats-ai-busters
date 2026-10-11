package introart

// RackLED is one status LED of the server racks; Color 3 is amber, 2 is green.
type RackLED struct{ X, Y, Color int }

// RackLEDs lists every status LED of a server room of the given width.
func RackLEDs(width int) []RackLED {
	var leds []RackLED
	for left := 7; left < width-52; left += 34 {
		for slot, y := 0, 21; y < 61; slot, y = slot+1, y+8 {
			for column, x := range [3]int{left + 17, left + 20, left + 23} {
				color := 3
				if (left+slot+column)%3 != 0 {
					color = 2
				}
				leds = append(leds, RackLED{x, y + 2, color})
				if column == 2 {
					c2 := 2
					if (left+slot)%2 != 0 {
						c2 = 3
					}
					leds = append(leds, RackLED{x, y + 4, c2})
				}
			}
		}
	}
	return leds
}

// ServerRoom draws the data-center aisle of the given width (160 or 256).
func ServerRoom(width int) *Canvas {
	art := NewCanvas(width)
	art.Rect(0, 70, width, Height, 0)
	art.Line(0, 72, width-1, 72, 2)
	for _, y := range []int{79, 87} {
		for x := 12 + y%11; x < width-18; x += 40 {
			art.Line(x, y, x+18, y, 1)
		}
	}
	for left := 7; left < width-52; left += 34 {
		art.Rect(left, 12, left+29, 70, 2)
		art.Rect(left+2, 14, left+27, 68, 1)
		art.Rect(left+5, 17, left+24, 63, 0)
		for y := 21; y < 61; y += 8 {
			art.Line(left+7, y, left+14, y, 2)
		}
	}
	for _, led := range RackLEDs(width) {
		art.Dot(led.X, led.Y, led.Color)
	}
	// the AI's wall terminal at the end of the aisle
	art.Rect(width-50, 8, width-7, 65, 2)
	art.Rect(width-47, 11, width-10, 62, 0)
	lengths := []int{30, 22, 34, 18, 26, 30, 14}
	for i, y := 0, 16; i < len(lengths); i, y = i+1, y+6 {
		art.Line(width-14-lengths[i], y, width-14, y, 2) // left-aligned once mirrored
	}
	art.Rect(width-18, 57, width-14, 60, 3)
	return art
}

// CatHead draws a cat head; natural selects the tabby palette.
func CatHead(art *Canvas, cx, cy, rx, ry int, natural bool) {
	border, fur, muzzle := 2, 1, 2
	if natural {
		border, fur, muzzle = 0, 2, 3
	}
	art.Triangle(cx-rx+3, cy-ry-12, cx-rx, cx-3, cy-8, border)
	art.Triangle(cx+rx-3, cy-ry-12, cx+3, cx+rx, cy-8, border)
	art.Triangle(cx-rx+5, cy-ry-7, cx-rx+4, cx-9, cy-10, fur)
	art.Triangle(cx+rx-5, cy-ry-7, cx+9, cx+rx-4, cy-10, fur)
	art.Ellipse(cx, cy, rx+2, ry+2, border)
	art.Ellipse(cx, cy, rx, ry, fur)
	if natural && rx > 15 {
		for _, stripe := range []int{-12, 0, 12} {
			art.Line(cx+stripe, cy-27, cx+stripe/2, cy-15, 0)
			art.Line(cx+stripe+1, cy-26, cx+stripe/2+1, cy-16, 0)
		}
		for _, side := range []int{-1, 1} {
			for _, offset := range []int{0, 7, 14} {
				art.Line(cx+side*26, cy-5+offset, cx+side*17, cy-1+offset, 0)
			}
		}
	}
	art.Ellipse(cx-11, cy+13, 9, 6, muzzle)
	art.Ellipse(cx+11, cy+13, 9, 6, muzzle)
	for _, side := range []int{-1, 1} {
		span := 18
		if rx > 15 {
			span = 32
		}
		art.Line(cx+side*14, cy+12, cx+side*span, cy+9, muzzle)
		art.Line(cx+side*15, cy+16, cx+side*span, cy+18, muzzle)
	}
	noseColor := 3
	if natural {
		noseColor = border
	}
	art.Triangle(cx, cy+9, cx-4, cx+4, cy+13, noseColor)
	art.Line(cx, cy+13, cx, cy+17, 0)
	art.Line(cx-4, cy+19, cx, cy+17, 0)
	art.Line(cx, cy+17, cx+4, cy+19, 0)
}

// RevealedRoom is the 256-wide panorama with the tabby sitting at the right end.
func RevealedRoom() *Canvas {
	art := ServerRoom(256)
	art.Ellipse(232, 93, 18, 12, 2)
	art.Triangle(224, 70, 222, 230, 81, 0)
	art.Triangle(240, 70, 234, 244, 81, 0)
	art.Triangle(224, 73, 225, 228, 79, 2)
	art.Triangle(240, 73, 236, 240, 79, 2)
	art.Ellipse(232, 82, 12, 10, 0)
	art.Ellipse(232, 82, 10, 8, 2)
	art.Line(232, 75, 232, 78, 0)
	for _, x := range []int{227, 237} {
		art.Ellipse(x, 81, 2, 2, 3)
		art.Dot(x, 81, 0)
	}
	art.Ellipse(229, 87, 3, 2, 3)
	art.Ellipse(235, 87, 3, 2, 3)
	art.Dot(232, 85, 0)
	for _, side := range []int{-1, 1} {
		art.Line(232+side*6, 87, 232+side*13, 85, 3)
	}
	return art
}

// FaceSplit: the DLI kernel switches fur/cream to rack green/amber at x=84..99.
const FaceSplit = 86

// Face is the mirrored close-up: the cat on the left, the lit data center on the right.
func Face() *Canvas {
	room := ServerRoom(160).Mirrored()
	room.Rect(99, 27, 141, 70, 1) // the AI's desk terminal, right of the split
	room.Rect(102, 30, 138, 65, 2)
	room.Rect(105, 33, 135, 62, 0)
	for _, l := range [][2]int{{37, 20}, {42, 16}, {47, 22}, {52, 12}} {
		room.Line(108, l[0], 108+l[1], l[0], 2)
	}
	room.Line(109, 56, 112, 59, 2)
	room.Line(115, 56, 112, 59, 2)
	room.Line(112, 59, 112, 62, 2)
	room.Line(118, 62, 125, 62, 2)
	room.Rect(115, 70, 126, 73, 2)
	for _, row := range room.Pixels { // left of the split, rack green would render as fur
		for x := 0; x < FaceSplit+14; x++ {
			if row[x] == 2 || (x >= FaceSplit-2 && row[x] == 3) {
				row[x] = 1
			}
		}
	}

	cat := NewCanvas(160)
	for _, row := range cat.Pixels {
		for x := range row {
			row[x] = Unset
		}
	}
	cat.Ellipse(113, 91, 37, 27, 2)
	CatHead(cat, 112, 54, 27, 30, true)
	for _, cx := range []int{101, 125} {
		cat.Ellipse(cx, 52, 9, 7, 3)
		cat.Rect(cx-1, 48, cx+3, 57, 0)
		cat.Dot(cx+5, 49, 3)
	}
	cat.Mirrored()
	for y := 0; y < Height; y++ {
		for x := 0; x < cat.Width; x++ {
			if cat.Pixels[y][x] != Unset {
				if x >= FaceSplit-2 {
					panic("introart: cat crosses the face split")
				}
				room.Pixels[y][x] = cat.Pixels[y][x]
			}
		}
	}
	return room
}
