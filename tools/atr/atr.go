// Package atr builds out/cats-ai-busters.atr: a DOS-less, self-booting Atari
// disk image (90K single density, 128-byte sectors, 720 sectors).
//
// Layout:
//
//	sector 1      hand-assembled 6502 boot loader (see AssembleLoader)
//	sectors 2..N  the K65 XEX's resident segment (0x1000..end), read straight
//	              into place by the boot loader
//	remaining     disk-streamed "chunks" (ExtraChunks), starting at
//	              FirstChunkSector, independent of the linked program's size.
//
// assets/disk-chunks.bin records, per chunk, a 5-byte entry: word
// start_sector, word sector_count, byte checksum (sum of the padded bytes mod 256).
package atr

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/michal-zuchowski/cats-ai-busters/tools/k65src"
)

// Chunk is one disk-streamed asset: the whole file, or Length bytes at Offset.
type Chunk struct {
	Name   string
	Path   string
	Offset int
	Length int // < 0: whole file
}

// Data returns the chunk's unpadded bytes (python path.read_bytes()[offset:offset+length]).
func (c Chunk) Data() []byte {
	b, err := os.ReadFile(c.Path)
	if err != nil {
		panic(err)
	}
	if c.Length < 0 {
		return b
	}
	lo := min(c.Offset, len(b))
	hi := min(c.Offset+c.Length, len(b))
	return b[lo:hi]
}

func asset(name string) string { return k65src.Path("assets", name) }

func whole(name, file string) Chunk { return Chunk{name, asset(file), 0, -1} }
func part(name, file string, off, n int) Chunk {
	return Chunk{name, asset(file), off, n}
}

// ExtraChunks returns the chunk list in disk order.
func ExtraChunks() []Chunk {
	cs := []Chunk{
		part("room0-art", "level-rooms.pic", 0, 960),
		part("room1-art", "level-rooms.pic", 960, 960),
		part("room2-art", "level-rooms.pic", 1920, 960),
		whole("cam-cfg", "level-cams.bin"),
		whole("cone-step", "cone-step.bin"),
	}
	for i := 0; i < 10; i++ {
		cs = append(cs, part(fmt.Sprintf("plat-room%d", i), "plat-rooms.bin", i*1024, 1024))
	}
	cs = append(cs, whole("death-cry", "plat-cry.pcm"), whole("cat-atlas", "cat-sprites.bin"),
		whole("title-music", "music.bin"))
	for i := 1; i <= 4; i++ {
		cs = append(cs, whole(fmt.Sprintf("level%d-music", i), fmt.Sprintf("music-level-%02d.bin", i)))
	}
	return append(cs,
		part("title-top", "title-screen.pic", 0, 4000),
		part("title-middle", "title-screen.pic", 4000, 960),
		part("title-bottom", "title-screen.pic", 4960, 2560),
		whole("ai-portrait", "ai-portrait.pic"),
	)
}

const (
	SectorSize       = 128
	TotalSectors     = 720    // 90K single-density disk
	BootSectors      = 1      // BRCNT: the loader fits in sector 1 alone
	BLDADR           = 0x0700 // where the OS loads the boot sector(s)
	PayloadAddr      = 0x1000 // -lowAddr from main.k65proj
	FirstDataSector  = BootSectors + 1
	ResidentCapacity = 0xA000 - PayloadAddr
	FirstChunkSector = FirstDataSector + ResidentCapacity/SectorSize
	DDEVIC, DUNIT    = 0x0300, 0x0301
	DCOMND, DSTATS   = 0x0302, 0x0303
	DBUFLO, DBUFHI   = 0x0304, 0x0305
	DTIMLO           = 0x0306
	DBYTLO, DBYTHI   = 0x0308, 0x0309
	DAUX1, DAUX2     = 0x030A, 0x030B
	SIOV             = 0xE459
	COUNTLO, COUNTHI = 0xCB, 0xCC // zero-page sector counter, free at boot time
)

// XEXPath, ATRPath and ChunksPath are the default inputs/outputs.
func XEXPath() string    { return k65src.Path("out", "cats-ai-busters.xex") }
func ATRPath() string    { return k65src.Path("out", "cats-ai-busters.atr") }
func ChunksPath() string { return k65src.Path("assets", "disk-chunks.bin") }

type op struct {
	size int
	make func(pc int) ([]byte, error)
}

// asm is a minimal two-pass 6502 assembler: just enough for the boot loader.
type asm struct {
	origin, pc int
	ops        []op
	labels     map[string]int
}

func (a *asm) emit(b ...byte) {
	a.ops = append(a.ops, op{len(b), func(int) ([]byte, error) { return b, nil }})
	a.pc += len(b)
}

func (a *asm) label(n string)         { a.labels[n] = a.pc }
func (a *asm) ldaImm(v int)           { a.emit(0xA9, byte(v)) }
func (a *asm) absOp(o byte, addr int) { a.emit(o, byte(addr), byte(addr>>8)) }
func (a *asm) staAbs(addr int)        { a.absOp(0x8D, addr) }
func (a *asm) ldaAbs(addr int)        { a.absOp(0xAD, addr) }
func (a *asm) adcImm(v int)           { a.emit(0x69, byte(v)) }
func (a *asm) incAbs(addr int)        { a.absOp(0xEE, addr) }
func (a *asm) staZP(addr int)         { a.emit(0x85, byte(addr)) }
func (a *asm) ldaZP(addr int)         { a.emit(0xA5, byte(addr)) }
func (a *asm) decZP(addr int)         { a.emit(0xC6, byte(addr)) }
func (a *asm) oraZP(addr int)         { a.emit(0x05, byte(addr)) }
func (a *asm) clc()                   { a.emit(0x18) }
func (a *asm) jsr(addr int)           { a.absOp(0x20, addr) }
func (a *asm) jmp(addr int)           { a.absOp(0x4C, addr) }

func (a *asm) bne(name string) {
	a.ops = append(a.ops, op{2, func(pc int) ([]byte, error) {
		off := a.labels[name] - (pc + 2)
		if off < -128 || off > 127 {
			return nil, fmt.Errorf("branch out of range: %s", name)
		}
		return []byte{0xD0, byte(off)}, nil
	}})
	a.pc += 2
}

func (a *asm) assemble() ([]byte, error) {
	var out []byte
	pc := a.origin
	for _, o := range a.ops {
		b, err := o.make(pc)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
		pc += o.size
	}
	return out, nil
}

// AssembleLoader assembles the boot loader reading sectorCount sectors from startSector.
func AssembleLoader(startSector, sectorCount, codeOrigin int) ([]byte, error) {
	a := &asm{origin: codeOrigin, pc: codeOrigin, labels: map[string]int{}}
	a.ldaImm(startSector & 0xFF)
	a.staAbs(DAUX1)
	a.ldaImm(startSector >> 8)
	a.staAbs(DAUX2)
	a.ldaImm(PayloadAddr & 0xFF)
	a.staAbs(DBUFLO)
	a.ldaImm(PayloadAddr >> 8)
	a.staAbs(DBUFHI)
	a.ldaImm(sectorCount & 0xFF)
	a.staZP(COUNTLO)
	a.ldaImm(sectorCount >> 8)
	a.staZP(COUNTHI)
	a.label("loop")
	a.ldaImm(0x31) // DDEVIC: disk
	a.staAbs(DDEVIC)
	a.ldaImm(0x01) // DUNIT: drive 1
	a.staAbs(DUNIT)
	a.ldaImm(0x52) // DCOMND: read sector ('R')
	a.staAbs(DCOMND)
	a.ldaImm(0x40) // DSTATS: peripheral -> memory
	a.staAbs(DSTATS)
	a.ldaImm(0x0F) // DTIMLO: 15s timeout
	a.staAbs(DTIMLO)
	a.ldaImm(SectorSize & 0xFF)
	a.staAbs(DBYTLO)
	a.ldaImm(SectorSize >> 8)
	a.staAbs(DBYTHI)
	a.jsr(SIOV)
	a.clc()
	a.ldaAbs(DBUFLO)
	a.adcImm(SectorSize)
	a.staAbs(DBUFLO)
	a.ldaAbs(DBUFHI)
	a.adcImm(0)
	a.staAbs(DBUFHI)
	a.incAbs(DAUX1)
	a.bne("nowrap")
	a.incAbs(DAUX2)
	a.label("nowrap")
	a.ldaZP(COUNTLO)
	a.bne("decl")
	a.decZP(COUNTHI)
	a.label("decl")
	a.decZP(COUNTLO)
	a.ldaZP(COUNTLO)
	a.oraZP(COUNTHI)
	a.bne("loop")
	a.jmp(PayloadAddr)
	return a.assemble()
}

// ParseXEXMainSegment returns the bytes of the 0x1000.. segment (the resident program).
func ParseXEXMainSegment(xex []byte) ([]byte, error) {
	i := 0
	for i < len(xex) {
		if i+2 <= len(xex) && xex[i] == 0xFF && xex[i+1] == 0xFF {
			i += 2
			continue
		}
		if i+4 > len(xex) {
			return nil, fmt.Errorf("truncated XEX segment header")
		}
		start := int(binary.LittleEndian.Uint16(xex[i:]))
		end := int(binary.LittleEndian.Uint16(xex[i+2:]))
		i += 4
		n := end - start + 1
		hi := min(i+n, len(xex))
		if start == PayloadAddr {
			return xex[i:hi], nil
		}
		i += n
	}
	return nil, fmt.Errorf("no segment starting at 0x%04X found in %s", PayloadAddr, XEXPath())
}

func padSectors(b []byte) []byte {
	n := (len(b) + SectorSize - 1) / SectorSize * SectorSize
	return append(b[:len(b):len(b)], make([]byte, n-len(b))...)
}

// BuildATR returns the ATR image, the resident sector count and the chunk table.
func BuildATR(payload []byte, chunks []Chunk) (image []byte, sectorCount int, table []byte, err error) {
	if len(payload) > ResidentCapacity {
		return nil, 0, nil, fmt.Errorf("resident program exceeds the reserved $1000..$9FFF bank")
	}
	payload = padSectors(payload)
	sectorCount = len(payload) / SectorSize

	codeOrigin := BLDADR + 6
	code, err := AssembleLoader(FirstDataSector, sectorCount, codeOrigin)
	if err != nil {
		return nil, 0, nil, err
	}
	if len(code) > SectorSize-6 {
		return nil, 0, nil, fmt.Errorf("boot loader (%d bytes) no longer fits in one sector", len(code))
	}
	body := make([]byte, TotalSectors*SectorSize)
	boot := body[:SectorSize]
	boot[0] = 0x00
	boot[1] = BootSectors
	binary.LittleEndian.PutUint16(boot[2:], BLDADR)
	binary.LittleEndian.PutUint16(boot[4:], uint16(codeOrigin))
	copy(boot[6:], code)
	copy(body[(FirstDataSector-1)*SectorSize:], payload)

	next := FirstChunkSector
	for _, c := range chunks {
		data := padSectors(c.Data())
		count := len(data) / SectorSize
		if next+count-1 > TotalSectors {
			return nil, 0, nil, fmt.Errorf("chunk %q does not fit on a 90K disk", c.Name)
		}
		copy(body[(next-1)*SectorSize:], data)
		sum := 0
		for _, v := range data {
			sum += int(v)
		}
		table = binary.LittleEndian.AppendUint16(table, uint16(next))
		table = binary.LittleEndian.AppendUint16(table, uint16(count))
		table = append(table, byte(sum))
		next += count
	}
	paragraphs := len(body) / 16
	header := make([]byte, 16)
	binary.LittleEndian.PutUint16(header[0:], 0x0296)
	binary.LittleEndian.PutUint16(header[2:], uint16(paragraphs&0xFFFF))
	binary.LittleEndian.PutUint16(header[4:], SectorSize)
	binary.LittleEndian.PutUint16(header[6:], uint16(paragraphs>>16))
	return append(header, body...), sectorCount, table, nil
}

// Run is the make-atr command: args are os.Args[1:]. It supports --table-only.
func Run(args []string) error {
	tableOnly := false
	for _, a := range args {
		switch a {
		case "--table-only", "-table-only":
			tableOnly = true
		case "-h", "--help", "-help":
			fmt.Println("usage: make-atr [--table-only]\nPack the Atari disk and its stable chunk table.\n" +
				"  --table-only  prepare chunk metadata without requiring or changing XEX/ATR")
			return nil
		default:
			return fmt.Errorf("unrecognized argument: %s", a)
		}
	}
	var payload []byte
	if tableOnly {
		payload = make([]byte, SectorSize)
	} else {
		xex, err := os.ReadFile(XEXPath())
		if err != nil {
			return fmt.Errorf("missing %s; build main.k65proj first", XEXPath())
		}
		if payload, err = ParseXEXMainSegment(xex); err != nil {
			return err
		}
	}
	chunks := ExtraChunks()
	image, sectorCount, table, err := BuildATR(payload, chunks)
	if err != nil {
		return err
	}
	if err := os.WriteFile(ChunksPath(), table, 0o644); err != nil {
		return err
	}
	if tableOnly {
		fmt.Printf("wrote %s (%d stable chunk entries; XEX/ATR unchanged)\n", ChunksPath(), len(chunks))
		return nil
	}
	if err := os.MkdirAll(k65src.Path("out"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(ATRPath(), image, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes, %d payload sectors from sector %d, %d extra chunk(s))\n",
		ATRPath(), len(image), sectorCount, FirstDataSector, len(chunks))
	return nil
}
