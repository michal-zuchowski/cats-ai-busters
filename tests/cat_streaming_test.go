package tests

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/atr"
	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// Cat-atlas disk/overlay checks using synthetic payloads, without compiling K65.

func TestCatStreamingSourceLayout(t *testing.T) {
	source := src()
	reveal := funcBody(source, "RevealImg")
	draw := funcBody(source, "draw_frame")
	atlas, panorama := readAsset("cat-sprites.bin"), readAsset("room.pic")

	ensure(t, !strings.Contains(source, `binary "assets/cat-sprites.bin"`) && !strings.Contains(source, "data CatSprites {"),
		"resident CatSprites data must be gone")
	ensure(t, strings.Contains(reveal, "align 4096") && strings.Contains(reveal, `binary "assets/room.pic"`),
		"RevealImg must be 4096-aligned and hold room.pic")
	ensure(t, len(atlas) == 5504 && len(atlas) <= len(panorama) && len(panorama) == 6144,
		"atlas %d / panorama %d sizes", len(atlas), len(panorama))
	ensure(t, strings.Contains(draw, "&>RevealImg") && strings.Contains(draw, "a=FrameMap,x"),
		"draw_frame must index RevealImg through FrameMap")
	ensure(t, regexp.MustCompile(`disk_dest=a=&<RevealImg\s+disk_dest\+1=a=&>RevealImg\s+a=16\s+load_room_chunk`).MatchString(funcBody(source, "load_level_assets")),
		"load_level_assets must load chunk 16 into RevealImg")

	init := funcBody(source, "init_gameplay")
	sw, load, pmg := mustIndex(t, init, "switch_to_term"), mustIndex(t, init, "load_level_assets"), mustIndex(t, init, "init_pmg")
	ensure(t, sw < load && load < pmg, "init_gameplay order: switch_to_term, load_level_assets, init_pmg")
	skip := mustIndex(t, source, "intro_skip:\n")
	if !strings.Contains(source[skip:], "  init_gameplay\n") {
		t.Fatal("init_gameplay not called after intro_skip")
	}

	proj := string(mustRead(k65src.Path("main.k65proj")))
	ensure(t, strings.Contains(proj, "-lowAddr 0x1000"), "main.k65proj lacks -lowAddr 0x1000")
	ensure(t, strings.Contains(proj, "-hiaddr 0x9FFF"), "main.k65proj lacks -hiaddr 0x9FFF")
}

func TestCatStreamingChunkTable(t *testing.T) {
	chunks := atr.ExtraChunks()
	ensure(t, chunks[15].Name == "death-cry" && chunks[16].Name == "cat-atlas",
		"chunks 15/16 = %q/%q", chunks[15].Name, chunks[16].Name)
	ensure(t, len(chunks) == 26, "%d chunks, want 26", len(chunks))
	ensure(t, atr.FirstChunkSector == 290, "FirstChunkSector = %d", atr.FirstChunkSector)

	// Program size changes must not invalidate the table embedded by the compiler.
	_, _, table, err := atr.BuildATR(make([]byte, 128), chunks)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{1024, atr.ResidentCapacity} {
		_, _, other, err := atr.BuildATR(make([]byte, size), chunks)
		if err != nil {
			t.Fatal(err)
		}
		ensure(t, bytes.Equal(table, other), "chunk table changes with resident size %d", size)
	}
	ensure(t, bytes.Equal(readAsset("disk-chunks.bin"), table), "assets/disk-chunks.bin differs from the built table")
	if _, _, _, err := atr.BuildATR(make([]byte, atr.ResidentCapacity+1), chunks); err == nil {
		t.Error("oversized resident program accepted")
	} else if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("oversized resident error %q lacks \"reserved\"", err)
	}
}

func TestCatStreamingDiskPayloads(t *testing.T) {
	chunks := atr.ExtraChunks()
	image, _, table, err := atr.BuildATR(make([]byte, 128), chunks)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range chunks {
		start := int(binary.LittleEndian.Uint16(table[i*5:]))
		count := int(binary.LittleEndian.Uint16(table[i*5+2:]))
		checksum := table[i*5+4]
		padded := make([]byte, count*atr.SectorSize)
		copy(padded, c.Data())
		disk := image[16+(start-1)*atr.SectorSize : 16+(start-1+count)*atr.SectorSize]
		sum := 0
		for _, v := range padded {
			sum += int(v)
		}
		ensure(t, bytes.Equal(disk, padded) && int(checksum) == sum&255, "chunk %s disk bytes/checksum mismatch", c.Name)
	}

	atlas, panorama, mapping := readAsset("cat-sprites.bin"), readAsset("room.pic"), readAsset("cat-frame-map.bin")
	start := int(binary.LittleEndian.Uint16(table[16*5:]))
	count := int(binary.LittleEndian.Uint16(table[16*5+2:]))
	ensure(t, count == 43, "cat-atlas chunk has %d sectors, want 43", count)
	streamed := image[16+(start-1)*128 : 16+(start-1+count)*128]
	memory := append([]byte{0xAA}, panorama...)
	memory = append(memory, 0xBB)
	copy(memory[1:], streamed)
	ensure(t, memory[0] == 0xAA && memory[len(memory)-1] == 0xBB, "streaming overran the RevealImg guards")
	ensure(t, bytes.Equal(memory[1+len(atlas):len(memory)-1], panorama[len(atlas):]),
		"streaming disturbed the panorama tail")
	for _, physical := range mapping {
		off := int(physical) * 128
		ensure(t, bytes.Equal(memory[1+off:1+off+128], atlas[off:off+128]), "physical frame %d not intact in memory", physical)
	}

	maxPhys := byte(0)
	for _, p := range mapping {
		maxPhys = max(maxPhys, p)
	}
	ensure(t, len(mapping) == 58 && maxPhys == 42, "map len %d max %d, want 58/42", len(mapping), maxPhys)
	// old logical prefix remains mapped byte-for-byte
	want, _ := hex.DecodeString("000102030405000102030405060708090a0b0c0d08090a0b0c0d0e0f101112131415" +
		"16061718191a1b1c1d0e1e1f2021222324252627")
	ensure(t, len(mapping) >= 54 && bytes.Equal(mapping[:54], want), "cat-frame-map prefix changed")
}
