package tests

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"

	"github.com/michal-zuchowski/cats-ai-busters/tools/atr"
)

// assets/disk-chunks.bin matches the ATR's on-disk chunk layout.
func TestDiskChunksMatchATR(t *testing.T) {
	atrBytes, err := os.ReadFile(atr.ATRPath())
	if err != nil {
		t.Skip("run go run ./tools/cmd/make-atr first (out/cats-ai-busters.atr missing)")
	}
	table := readAsset("disk-chunks.bin")
	chunks := atr.ExtraChunks()
	if len(table) != 5*len(chunks) {
		t.Fatalf("5 bytes (2+2+1) per chunk entry: table %d bytes, %d chunks", len(table), len(chunks))
	}
	body := atrBytes[16:]
	for i, c := range chunks {
		data := c.Data()
		start := int(binary.LittleEndian.Uint16(table[i*5:]))
		count := int(binary.LittleEndian.Uint16(table[i*5+2:]))
		checksum := table[i*5+4]
		paddedLen := (len(data) + atr.SectorSize - 1) / atr.SectorSize * atr.SectorSize
		padded := append(append([]byte{}, data...), make([]byte, paddedLen-len(data))...)
		if count != paddedLen/atr.SectorSize {
			t.Errorf("%s: sector count %d, want %d", c.Name, count, paddedLen/atr.SectorSize)
			continue
		}
		onDisk := body[(start-1)*atr.SectorSize : (start-1+count)*atr.SectorSize]
		if !bytes.Equal(onDisk, padded) {
			t.Errorf("chunk %q bytes on disk must match %s", c.Name, c.Path)
		}
		sum := 0
		for _, v := range padded {
			sum += int(v)
		}
		if checksum != byte(sum) {
			t.Errorf("chunk %q checksum mismatch", c.Name)
		}
	}
}
