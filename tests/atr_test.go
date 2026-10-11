package tests

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/atr"
)

// ATR layout: header, boot sector, and payload sectors of out/cats-ai-busters.atr.
func TestATRLayout(t *testing.T) {
	xexBytes, err := os.ReadFile(atr.XEXPath())
	if err != nil {
		t.Skip("build main.k65proj first (out/cats-ai-busters.xex missing)")
	}
	atrBytes, err := os.ReadFile(atr.ATRPath())
	if err != nil {
		t.Skip("run go run ./tools/cmd/make-atr first (out/cats-ai-busters.atr missing)")
	}
	payload, err := atr.ParseXEXMainSegment(xexBytes)
	if err != nil {
		t.Fatal(err)
	}

	le := binary.LittleEndian
	if magic := le.Uint16(atrBytes[0:]); magic != 0x0296 {
		t.Fatalf("bad ATR magic %#x", magic)
	}
	if ss := le.Uint16(atrBytes[4:]); ss != 128 {
		t.Fatalf("sector size %d", ss)
	}
	body := atrBytes[16:]
	if len(body) != atr.TotalSectors*atr.SectorSize {
		t.Fatalf("expected a 90K image, got %d body bytes", len(body))
	}
	paragraphs := int(le.Uint16(atrBytes[2:])) | int(le.Uint16(atrBytes[6:]))<<16
	if paragraphs*16 != len(body) {
		t.Fatalf("paragraph count %d does not match body size %d", paragraphs, len(body))
	}
	sector := func(n int) []byte { // 1-based
		off := (n - 1) * atr.SectorSize
		return body[off : off+atr.SectorSize]
	}

	boot := sector(1)
	if boot[0] != 0 {
		t.Errorf("boot flag must mark the disk bootable, got %d", boot[0])
	}
	if int(boot[1]) != atr.BootSectors {
		t.Errorf("BRCNT = %d", boot[1])
	}
	bldadr, binitad := le.Uint16(boot[2:]), le.Uint16(boot[4:])
	if bldadr != 0x0700 {
		t.Errorf("BLDADR = %#x", bldadr)
	}
	if int(binitad) != atr.BLDADR+6 {
		t.Errorf("loader code must start right after the 6-byte header, BINITAD = %#x", binitad)
	}

	code := boot[6:]
	jmp := []byte{0x4C, atr.PayloadAddr & 0xFF, atr.PayloadAddr >> 8}
	if i := bytes.Index(code, jmp); i <= 0 {
		t.Errorf("loader must JMP to the payload address (index %d)", i)
	}
	for _, needed := range [][]byte{{0xA9, 0x31}, {0xA9, 0x52}, {0x20, atr.SIOV & 0xFF, atr.SIOV >> 8}} {
		if !bytes.Contains(code, needed) {
			t.Errorf("loader is missing % X", needed)
		}
	}

	sectorCount := (len(payload) + atr.SectorSize - 1) / atr.SectorSize
	padded := append(append([]byte{}, payload...), make([]byte, sectorCount*atr.SectorSize-len(payload))...)
	first := atr.FirstDataSector
	if first != atr.BootSectors+1 {
		t.Fatalf("first data sector %d", first)
	}
	var onDisk []byte
	for n := 0; n < sectorCount; n++ {
		onDisk = append(onDisk, sector(first+n)...)
	}
	if !bytes.Equal(onDisk, padded) {
		t.Errorf("disk payload must equal the XEX's 0x1000.. segment")
	}
}
