package sim

import "testing"

// run loads code at $0200, executes n instructions and returns the CPU.
func run(code []byte, n int, setup func(*CPU)) *CPU {
	c := NewCPU()
	copy(c.Mem[0x200:], code)
	c.PC = 0x200
	if setup != nil {
		setup(c)
	}
	for i := 0; i < n; i++ {
		c.Step()
	}
	return c
}

func TestADCSBCBinary(t *testing.T) {
	c := run([]byte{0x18, 0xA9, 0x50, 0x69, 0x50}, 3, nil)
	if c.A != 0xA0 || c.P&FlagV == 0 || c.P&FlagN == 0 || c.P&FlagC != 0 {
		t.Errorf("0x50+0x50: A=%02x P=%02x", c.A, c.P)
	}
	c = run([]byte{0x38, 0xA9, 0x50, 0xE9, 0xF0}, 3, nil)
	if c.A != 0x60 || c.P&FlagC != 0 || c.P&FlagV != 0 {
		t.Errorf("0x50-0xF0: A=%02x P=%02x", c.A, c.P)
	}
	c = run([]byte{0x38, 0xA9, 0x05, 0xE9, 0x05}, 3, nil)
	if c.A != 0 || c.P&FlagZ == 0 || c.P&FlagC == 0 {
		t.Errorf("5-5: A=%02x P=%02x", c.A, c.P)
	}
}

func TestADCSBCDecimal(t *testing.T) {
	// SED CLC LDA #$58 ADC #$46 -> $04 carry set
	c := run([]byte{0xF8, 0x18, 0xA9, 0x58, 0x69, 0x46}, 4, nil)
	if c.A != 0x04 || c.P&FlagC == 0 {
		t.Errorf("58+46: A=%02x P=%02x", c.A, c.P)
	}
	// 12 + 34 + carry
	c = run([]byte{0xF8, 0x38, 0xA9, 0x12, 0x69, 0x34}, 4, nil)
	if c.A != 0x47 || c.P&FlagC != 0 {
		t.Errorf("12+34+1: A=%02x P=%02x", c.A, c.P)
	}
	// 46 - 12 = 34 (carry set = no borrow)
	c = run([]byte{0xF8, 0x38, 0xA9, 0x46, 0xE9, 0x12}, 4, nil)
	if c.A != 0x34 || c.P&FlagC == 0 {
		t.Errorf("46-12: A=%02x P=%02x", c.A, c.P)
	}
	// 40 - 13 = 27
	c = run([]byte{0xF8, 0x38, 0xA9, 0x40, 0xE9, 0x13}, 4, nil)
	if c.A != 0x27 || c.P&FlagC == 0 {
		t.Errorf("40-13: A=%02x P=%02x", c.A, c.P)
	}
	// 12 - 21 = 91 with borrow (carry clear)
	c = run([]byte{0xF8, 0x38, 0xA9, 0x12, 0xE9, 0x21}, 4, nil)
	if c.A != 0x91 || c.P&FlagC != 0 {
		t.Errorf("12-21: A=%02x P=%02x", c.A, c.P)
	}
	// 99 + 01 = 00 carry; py65 (NMOS) sets Z from the undecimal ALU result ($9A), so Z stays clear
	c = run([]byte{0xF8, 0x18, 0xA9, 0x99, 0x69, 0x01}, 4, nil)
	if c.A != 0 || c.P&FlagC == 0 || c.P&FlagZ != 0 || c.P&FlagN == 0 {
		t.Errorf("99+01: A=%02x P=%02x", c.A, c.P)
	}
}

func TestBranchCycles(t *testing.T) {
	// not taken: 2 cycles
	c := run([]byte{0xF0, 0x10}, 1, nil)
	if c.Cycles != 2 || c.PC != 0x202 {
		t.Errorf("not taken: cycles=%d pc=%04x", c.Cycles, c.PC)
	}
	// taken, same page: 3 cycles
	c = run([]byte{0xD0, 0x10}, 1, nil)
	if c.Cycles != 3 || c.PC != 0x212 {
		t.Errorf("taken: cycles=%d pc=%04x", c.Cycles, c.PC)
	}
	// taken forward across a page: 4 cycles
	c = run([]byte{0xD0, 0x7F}, 1, func(c *CPU) { c.PC = 0x2F0; c.Mem[0x2F0], c.Mem[0x2F1] = 0xD0, 0x7F })
	if c.Cycles != 4 || c.PC != 0x371 {
		t.Errorf("page cross: cycles=%d pc=%04x", c.Cycles, c.PC)
	}
	// taken backward across a page: 4 cycles
	c = run(nil, 1, func(c *CPU) { c.PC = 0x300; c.Mem[0x300], c.Mem[0x301] = 0xD0, 0xFC })
	if c.Cycles != 4 || c.PC != 0x2FE {
		t.Errorf("backward cross: cycles=%d pc=%04x", c.Cycles, c.PC)
	}
}

func TestIndexedPageCrossCycles(t *testing.T) {
	c := run([]byte{0xBD, 0xFF, 0x20}, 1, func(c *CPU) { c.X = 1 })
	if c.Cycles != 5 {
		t.Errorf("LDA abs,X cross: %d", c.Cycles)
	}
	c = run([]byte{0xBD, 0x00, 0x20}, 1, func(c *CPU) { c.X = 1 })
	if c.Cycles != 4 {
		t.Errorf("LDA abs,X no cross: %d", c.Cycles)
	}
	c = run([]byte{0x9D, 0xFF, 0x20}, 1, func(c *CPU) { c.X = 1 })
	if c.Cycles != 5 {
		t.Errorf("STA abs,X: %d", c.Cycles)
	}
	c = run([]byte{0xB1, 0x10}, 1, func(c *CPU) { c.Mem[0x10], c.Mem[0x11], c.Y = 0xFF, 0x20, 1 })
	if c.Cycles != 6 {
		t.Errorf("LDA (zp),Y cross: %d", c.Cycles)
	}
	c = run([]byte{0xCE, 0x00, 0x20}, 1, nil)
	if c.Cycles != 3 { // py65 quirk
		t.Errorf("DEC abs cycles: %d", c.Cycles)
	}
}

func TestJSRRTSBRKRTI(t *testing.T) {
	c := run([]byte{0x20, 0x00, 0x03}, 1, func(c *CPU) { c.Mem[0x300] = 0x60 })
	if c.PC != 0x300 || c.SP != 0xFD || c.Mem[0x1FF] != 0x02 || c.Mem[0x1FE] != 0x02 || c.Cycles != 6 {
		t.Errorf("JSR: pc=%04x sp=%02x stack=%02x%02x cyc=%d", c.PC, c.SP, c.Mem[0x1FF], c.Mem[0x1FE], c.Cycles)
	}
	c.Step()
	if c.PC != 0x203 || c.SP != 0xFF || c.Cycles != 12 {
		t.Errorf("RTS: pc=%04x sp=%02x cyc=%d", c.PC, c.SP, c.Cycles)
	}
	// BRK pushes PC+2 and P|B|U, sets I, vectors via $FFFE; RTI restores.
	c = run([]byte{0x00}, 1, func(c *CPU) {
		c.Mem[0xFFFE], c.Mem[0xFFFF] = 0x00, 0x04
		c.Mem[0x400] = 0x40
		c.P = FlagU | FlagC
	})
	if c.PC != 0x400 || c.P&FlagI == 0 || c.SP != 0xFC || c.Mem[0x1FF] != 0x02 || c.Mem[0x1FE] != 0x02 ||
		c.Mem[0x1FD] != FlagU|FlagB|FlagC || c.Cycles != 7 {
		t.Errorf("BRK: pc=%04x p=%02x sp=%02x stack=%02x %02x %02x", c.PC, c.P, c.SP, c.Mem[0x1FF], c.Mem[0x1FE], c.Mem[0x1FD])
	}
	c.Step()
	if c.PC != 0x202 || c.SP != 0xFF || c.P != FlagU|FlagB|FlagC || c.Cycles != 13 {
		t.Errorf("RTI: pc=%04x p=%02x sp=%02x cyc=%d", c.PC, c.P, c.SP, c.Cycles)
	}
}

func TestStackWrap(t *testing.T) {
	c := run([]byte{0x48, 0x48}, 2, func(c *CPU) { c.SP = 0x00; c.A = 0x77 })
	if c.SP != 0xFE || c.Mem[0x100] != 0x77 || c.Mem[0x1FF] != 0x77 {
		t.Errorf("PHA wrap: sp=%02x", c.SP)
	}
	c = run([]byte{0x68}, 1, func(c *CPU) { c.SP = 0xFF; c.Mem[0x100] = 0x80 })
	if c.SP != 0 || c.A != 0x80 || c.P&FlagN == 0 {
		t.Errorf("PLA wrap: sp=%02x a=%02x", c.SP, c.A)
	}
	// JSR with SP=$00: high byte at $100, low byte at $1FF
	c = run([]byte{0x20, 0x00, 0x03}, 1, func(c *CPU) { c.SP = 0x00 })
	if c.SP != 0xFE || c.Mem[0x100] != 0x02 || c.Mem[0x1FF] != 0x02 {
		t.Errorf("JSR wrap: sp=%02x", c.SP)
	}
}

func TestAddressingQuirks(t *testing.T) {
	// JMP ($02FF) fetches the high byte from $0200
	c := run([]byte{0x6C, 0xFF, 0x02}, 1, func(c *CPU) { c.Mem[0x2FF] = 0x34 })
	if c.PC != 0x6C34 { // $0200 holds the JMP opcode itself
		t.Errorf("JMP ind wrap: pc=%04x", c.PC)
	}
	// LDA ($FF,X) with X=0 wraps the pointer inside the zero page
	c = run([]byte{0xA1, 0xFF}, 1, func(c *CPU) { c.Mem[0xFF], c.Mem[0x00], c.Mem[0x1234] = 0x34, 0x12, 0x99 })
	if c.A != 0x99 {
		t.Errorf("(zp,X) wrap: a=%02x", c.A)
	}
	// LDA $FF,X wraps within zero page
	c = run([]byte{0xB5, 0xFF}, 1, func(c *CPU) { c.X = 2; c.Mem[1] = 0x42 })
	if c.A != 0x42 {
		t.Errorf("zp,X wrap: a=%02x", c.A)
	}
}

func TestUndocumentedOpcodesAreZeroCycleNOPs(t *testing.T) {
	c := run([]byte{0x02, 0xFF, 0xEA}, 1, nil)
	if c.PC != 0x202 || c.Cycles != 0 {
		t.Errorf("unknown opcode: pc=%04x cycles=%d", c.PC, c.Cycles)
	}
}

func TestShiftsFlagsAndTransfers(t *testing.T) {
	c := run([]byte{0x38, 0xA9, 0x81, 0x2A}, 3, nil) // SEC LDA #$81 ROL A
	if c.A != 0x03 || c.P&FlagC == 0 {
		t.Errorf("ROL: a=%02x p=%02x", c.A, c.P)
	}
	c = run([]byte{0x38, 0xA9, 0x01, 0x6A}, 3, nil) // ROR A
	if c.A != 0x80 || c.P&FlagC == 0 || c.P&FlagN == 0 {
		t.Errorf("ROR: a=%02x p=%02x", c.A, c.P)
	}
	c = run([]byte{0xA9, 0x81, 0x4A}, 2, nil) // LSR
	if c.A != 0x40 || c.P&FlagC == 0 || c.P&FlagN != 0 {
		t.Errorf("LSR: a=%02x p=%02x", c.A, c.P)
	}
	c = run([]byte{0xA9, 0xC0, 0x85, 0x10, 0x24, 0x10}, 3, nil) // BIT
	if c.P&FlagN == 0 || c.P&FlagV == 0 || c.P&FlagZ != 0 {
		t.Errorf("BIT: p=%02x", c.P)
	}
	c = run([]byte{0xA9, 0x05, 0xC9, 0x05}, 2, nil) // CMP
	if c.P&(FlagZ|FlagC) != FlagZ|FlagC {
		t.Errorf("CMP: p=%02x", c.P)
	}
	c = run([]byte{0x08, 0x68}, 2, func(c *CPU) { c.P = FlagC | FlagU }) // PHP PLA
	if c.A != FlagC|FlagU|FlagB {
		t.Errorf("PHP: a=%02x", c.A)
	}
}
