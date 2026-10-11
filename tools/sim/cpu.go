package sim

// CPU is an NMOS 6502 core that reproduces py65's MPU6502 exactly: same flag
// handling (including its decimal-mode ADC/SBC), cycle counts and quirks:
// DEC abs costs 3 cycles, undocumented opcodes are zero-cycle two-byte NOPs,
// conditional branches cost 3/4 cycles when taken, indexed reads add a cycle on
// a page cross, and JMP (ind) wraps within the page. Memory is plain RAM.
type CPU struct {
	Mem    [0x10000]byte
	PC     uint16
	A, X   byte
	Y, SP  byte
	P      byte
	Cycles int64
}

// Processor flags.
const (
	FlagC = 1
	FlagZ = 2
	FlagI = 4
	FlagD = 8
	FlagB = 16
	FlagU = 32
	FlagV = 64
	FlagN = 128
)

const (
	vecNMI = 0xFFFA
	vecIRQ = 0xFFFE
)

type mode uint8

const (
	imp mode = iota
	acc
	imm
	zpg
	zpx
	zpy
	inx
	iny
	abs
	abx
	aby
	ind
	rel
)

type opInfo struct {
	name   string
	mode   mode
	cycles uint8
	extra  bool // +1 cycle on page cross (read forms)
	ok     bool
}

var table [256]opInfo

func init() {
	type e struct {
		op     byte
		name   string
		m      mode
		cycles uint8
		extra  bool
	}
	for _, x := range []e{
		{0x00, "BRK", imp, 7, false}, {0x01, "ORA", inx, 6, false}, {0x05, "ORA", zpg, 3, false},
		{0x06, "ASL", zpg, 5, false}, {0x08, "PHP", imp, 3, false}, {0x09, "ORA", imm, 2, false},
		{0x0a, "ASL", acc, 2, false}, {0x0d, "ORA", abs, 4, false}, {0x0e, "ASL", abs, 6, false},
		{0x10, "BPL", rel, 2, false}, {0x11, "ORA", iny, 5, true}, {0x15, "ORA", zpx, 4, false},
		{0x16, "ASL", zpx, 6, false}, {0x18, "CLC", imp, 2, false}, {0x19, "ORA", aby, 4, true},
		{0x1d, "ORA", abx, 4, true}, {0x1e, "ASL", abx, 7, false},
		{0x20, "JSR", abs, 6, false}, {0x21, "AND", inx, 6, false}, {0x24, "BIT", zpg, 3, false},
		{0x25, "AND", zpg, 3, false}, {0x26, "ROL", zpg, 5, false}, {0x28, "PLP", imp, 4, false},
		{0x29, "AND", imm, 2, false}, {0x2a, "ROL", acc, 2, false}, {0x2c, "BIT", abs, 4, false},
		{0x2d, "AND", abs, 4, false}, {0x2e, "ROL", abs, 6, false},
		{0x30, "BMI", rel, 2, false}, {0x31, "AND", iny, 5, true}, {0x35, "AND", zpx, 4, false},
		{0x36, "ROL", zpx, 6, false}, {0x38, "SEC", imp, 2, false}, {0x39, "AND", aby, 4, true},
		{0x3d, "AND", abx, 4, true}, {0x3e, "ROL", abx, 7, false},
		{0x40, "RTI", imp, 6, false}, {0x41, "EOR", inx, 6, false}, {0x45, "EOR", zpg, 3, false},
		{0x46, "LSR", zpg, 5, false}, {0x48, "PHA", imp, 3, false}, {0x49, "EOR", imm, 2, false},
		{0x4a, "LSR", acc, 2, false}, {0x4c, "JMP", abs, 3, false}, {0x4d, "EOR", abs, 4, false},
		{0x4e, "LSR", abs, 6, false},
		{0x50, "BVC", rel, 2, false}, {0x51, "EOR", iny, 5, true}, {0x55, "EOR", zpx, 4, false},
		{0x56, "LSR", zpx, 6, false}, {0x58, "CLI", imp, 2, false}, {0x59, "EOR", aby, 4, true},
		{0x5d, "EOR", abx, 4, true}, {0x5e, "LSR", abx, 7, false},
		{0x60, "RTS", imp, 6, false}, {0x61, "ADC", inx, 6, false}, {0x65, "ADC", zpg, 3, false},
		{0x66, "ROR", zpg, 5, false}, {0x68, "PLA", imp, 4, false}, {0x69, "ADC", imm, 2, false},
		{0x6a, "ROR", acc, 2, false}, {0x6c, "JMP", ind, 5, false}, {0x6d, "ADC", abs, 4, false},
		{0x6e, "ROR", abs, 6, false},
		{0x70, "BVS", rel, 2, false}, {0x71, "ADC", iny, 5, true}, {0x75, "ADC", zpx, 4, false},
		{0x76, "ROR", zpx, 6, false}, {0x78, "SEI", imp, 2, false}, {0x79, "ADC", aby, 4, true},
		{0x7d, "ADC", abx, 4, true}, {0x7e, "ROR", abx, 7, false},
		{0x81, "STA", inx, 6, false}, {0x84, "STY", zpg, 3, false}, {0x85, "STA", zpg, 3, false},
		{0x86, "STX", zpg, 3, false}, {0x88, "DEY", imp, 2, false}, {0x8a, "TXA", imp, 2, false},
		{0x8c, "STY", abs, 4, false}, {0x8d, "STA", abs, 4, false}, {0x8e, "STX", abs, 4, false},
		{0x90, "BCC", rel, 2, false}, {0x91, "STA", iny, 6, false}, {0x94, "STY", zpx, 4, false},
		{0x95, "STA", zpx, 4, false}, {0x96, "STX", zpy, 4, false}, {0x98, "TYA", imp, 2, false},
		{0x99, "STA", aby, 5, false}, {0x9a, "TXS", imp, 2, false}, {0x9d, "STA", abx, 5, false},
		{0xa0, "LDY", imm, 2, false}, {0xa1, "LDA", inx, 6, false}, {0xa2, "LDX", imm, 2, false},
		{0xa4, "LDY", zpg, 3, false}, {0xa5, "LDA", zpg, 3, false}, {0xa6, "LDX", zpg, 3, false},
		{0xa8, "TAY", imp, 2, false}, {0xa9, "LDA", imm, 2, false}, {0xaa, "TAX", imp, 2, false},
		{0xac, "LDY", abs, 4, false}, {0xad, "LDA", abs, 4, false}, {0xae, "LDX", abs, 4, false},
		{0xb0, "BCS", rel, 2, false}, {0xb1, "LDA", iny, 5, true}, {0xb4, "LDY", zpx, 4, false},
		{0xb5, "LDA", zpx, 4, false}, {0xb6, "LDX", zpy, 4, false}, {0xb8, "CLV", imp, 2, false},
		{0xb9, "LDA", aby, 4, true}, {0xba, "TSX", imp, 2, false}, {0xbc, "LDY", abx, 4, true},
		{0xbd, "LDA", abx, 4, true}, {0xbe, "LDX", aby, 4, true},
		{0xc0, "CPY", imm, 2, false}, {0xc1, "CMP", inx, 6, false}, {0xc4, "CPY", zpg, 3, false},
		{0xc5, "CMP", zpg, 3, false}, {0xc6, "DEC", zpg, 5, false}, {0xc8, "INY", imp, 2, false},
		{0xc9, "CMP", imm, 2, false}, {0xca, "DEX", imp, 2, false}, {0xcc, "CPY", abs, 4, false},
		{0xcd, "CMP", abs, 4, false}, {0xce, "DEC", abs, 3, false},
		{0xd0, "BNE", rel, 2, false}, {0xd1, "CMP", iny, 5, true}, {0xd5, "CMP", zpx, 4, false},
		{0xd6, "DEC", zpx, 6, false}, {0xd8, "CLD", imp, 2, false}, {0xd9, "CMP", aby, 4, true},
		{0xdd, "CMP", abx, 4, true}, {0xde, "DEC", abx, 7, false},
		{0xe0, "CPX", imm, 2, false}, {0xe1, "SBC", inx, 6, false}, {0xe4, "CPX", zpg, 3, false},
		{0xe5, "SBC", zpg, 3, false}, {0xe6, "INC", zpg, 5, false}, {0xe8, "INX", imp, 2, false},
		{0xe9, "SBC", imm, 2, false}, {0xea, "NOP", imp, 2, false}, {0xec, "CPX", abs, 4, false},
		{0xed, "SBC", abs, 4, false}, {0xee, "INC", abs, 6, false},
		{0xf0, "BEQ", rel, 2, false}, {0xf1, "SBC", iny, 5, true}, {0xf5, "SBC", zpx, 4, false},
		{0xf6, "INC", zpx, 6, false}, {0xf8, "SED", imp, 2, false}, {0xf9, "SBC", aby, 4, true},
		{0xfd, "SBC", abx, 4, true}, {0xfe, "INC", abx, 7, false},
	} {
		table[x.op] = opInfo{x.name, x.m, x.cycles, x.extra, true}
	}
}

// NewCPU returns a CPU in py65's reset state (PC 0, SP $FF, P = B|U).
func NewCPU() *CPU {
	c := &CPU{}
	c.Reset()
	return c
}

func (c *CPU) Reset() {
	c.PC, c.SP, c.A, c.X, c.Y = 0, 0xFF, 0, 0, 0
	c.P = FlagB | FlagU
	c.Cycles = 0
}

func (c *CPU) word(addr uint16) uint16 {
	return uint16(c.Mem[addr]) | uint16(c.Mem[addr+1])<<8
}

// wrapWord reads a word whose high byte wraps within the page.
func (c *CPU) wrapWord(addr uint16) uint16 {
	return uint16(c.Mem[addr]) | uint16(c.Mem[addr&0xFF00+(addr+1)&0xFF])<<8
}

func (c *CPU) push(v byte) {
	c.Mem[0x100+uint16(c.SP)] = v
	c.SP--
}

func (c *CPU) pop() byte {
	c.SP++
	return c.Mem[0x100+uint16(c.SP)]
}

func (c *CPU) pushWord(v uint16) {
	c.push(byte(v >> 8))
	c.push(byte(v))
}

func (c *CPU) popWord() uint16 {
	lo := uint16(c.pop())
	return lo | uint16(c.pop())<<8
}

func (c *CPU) nz(v byte) {
	c.P &^= FlagZ | FlagN
	if v == 0 {
		c.P |= FlagZ
	} else {
		c.P |= v & FlagN
	}
}

// addr resolves the operand address with PC at the operand; extra counts page-cross cycles.
func (c *CPU) addr(m mode, extra bool, ex *int64) uint16 {
	pc := c.PC
	switch m {
	case imm:
		return pc
	case zpg:
		return uint16(c.Mem[pc])
	case zpx:
		return uint16(c.Mem[pc] + c.X)
	case zpy:
		return uint16(c.Mem[pc] + c.Y)
	case inx:
		return c.wrapWord(uint16(c.Mem[pc] + c.X))
	case iny:
		a1 := c.wrapWord(uint16(c.Mem[pc]))
		a2 := a1 + uint16(c.Y)
		if extra && a1&0xFF00 != a2&0xFF00 {
			*ex++
		}
		return a2
	case abs:
		return c.word(pc)
	case abx, aby:
		idx := c.X
		if m == aby {
			idx = c.Y
		}
		a1 := c.word(pc)
		a2 := a1 + uint16(idx)
		if extra && a1&0xFF00 != a2&0xFF00 {
			*ex++
		}
		return a2
	}
	panic("bad addressing mode")
}

func size(m mode) uint16 {
	switch m {
	case imp, acc:
		return 0
	case abs, abx, aby, ind:
		return 2
	}
	return 1
}

func (c *CPU) cmp(r, v byte) {
	c.P &^= FlagC | FlagZ | FlagN
	if r == v {
		c.P |= FlagC | FlagZ
	} else if r > v {
		c.P |= FlagC
	}
	c.P |= (r - v) & FlagN
}

func (c *CPU) adc(data byte) {
	a := c.A
	if c.P&FlagD != 0 {
		var half, dec, adj0, adj1 int
		n0 := int(data&0xf) + int(a&0xf) + int(c.P&FlagC)
		if n0 > 9 {
			adj0, half = 6, 1
		}
		n1 := int(data>>4) + int(a>>4) + half
		if n1 > 9 {
			adj1, dec = 6, 1
		}
		n0 &= 0xf
		n1 &= 0xf
		alu := byte(n1<<4 + n0)
		n0 = (n0 + adj0) & 0xf
		n1 = (n1 + adj1) & 0xf
		c.P &^= FlagC | FlagV | FlagN | FlagZ
		if alu == 0 {
			c.P |= FlagZ
		} else {
			c.P |= alu & FlagN
		}
		if dec == 1 {
			c.P |= FlagC
		}
		if ^(a^data)&(a^alu)&FlagN != 0 {
			c.P |= FlagV
		}
		c.A = byte(n1<<4 + n0)
		return
	}
	res := int(data) + int(a) + int(c.P&FlagC)
	c.P &^= FlagC | FlagV | FlagN | FlagZ
	if ^(a^data)&(a^byte(res))&FlagN != 0 {
		c.P |= FlagV
	}
	if res > 0xFF {
		c.P |= FlagC
	}
	r := byte(res)
	if r == 0 {
		c.P |= FlagZ
	} else {
		c.P |= r & FlagN
	}
	c.A = r
}

func (c *CPU) sbc(data byte) {
	a := c.A
	if c.P&FlagD != 0 {
		half, dec, adj0, adj1 := 1, 0, 0, 0
		n0 := int(a&0xf) + int(^data&0xf) + int(c.P&FlagC)
		if n0 <= 0xf {
			half, adj0 = 0, 10
		}
		n1 := int(a>>4) + int(^data>>4) + half
		if n1 <= 0xf {
			adj1 = 10 << 4
		}
		alu := int(a) + int(^data) + int(c.P&FlagC)
		if alu > 0xFF {
			dec = 1
		}
		alu &= 0xFF
		n0 = (alu + adj0) & 0xf
		n1 = ((alu + adj1) >> 4) & 0xf
		c.P &^= FlagC | FlagZ | FlagN | FlagV
		if alu == 0 {
			c.P |= FlagZ
		} else {
			c.P |= byte(alu) & FlagN
		}
		if dec == 1 {
			c.P |= FlagC
		}
		if (a^data)&(a^byte(alu))&FlagN != 0 {
			c.P |= FlagV
		}
		c.A = byte(n1<<4 + n0)
		return
	}
	res := int(a) + int(^data) + int(c.P&FlagC)
	c.P &^= FlagC | FlagZ | FlagV | FlagN
	if (a^data)&(a^byte(res))&FlagN != 0 {
		c.P |= FlagV
	}
	r := byte(res)
	if r == 0 {
		c.P |= FlagZ
	}
	if res > 0xFF {
		c.P |= FlagC
	}
	c.P |= r & FlagN
	c.A = r
}

// Step executes one instruction.
func (c *CPU) Step() {
	opc := c.Mem[c.PC]
	c.PC++
	info := &table[opc]
	if !info.ok {
		c.PC++
		return
	}
	var ex int64
	m := info.mode
	switch info.name {
	case "BRK":
		c.pushWord(c.PC + 1)
		c.P |= FlagB
		c.push(c.P | FlagB | FlagU)
		c.P |= FlagI
		c.PC = c.word(vecIRQ)
	case "JSR":
		c.pushWord(c.PC + 1)
		c.PC = c.word(c.PC)
	case "JMP":
		if m == ind {
			c.PC = c.wrapWord(c.word(c.PC))
		} else {
			c.PC = c.word(c.PC)
		}
	case "RTS":
		c.PC = c.popWord() + 1
	case "RTI":
		c.P = c.pop() | FlagB | FlagU
		c.PC = c.popWord()
	case "BPL", "BMI", "BVC", "BVS", "BCC", "BCS", "BNE", "BEQ":
		var flag byte
		var want bool
		switch info.name {
		case "BPL":
			flag, want = FlagN, false
		case "BMI":
			flag, want = FlagN, true
		case "BVC":
			flag, want = FlagV, false
		case "BVS":
			flag, want = FlagV, true
		case "BCC":
			flag, want = FlagC, false
		case "BCS":
			flag, want = FlagC, true
		case "BNE":
			flag, want = FlagZ, false
		default:
			flag, want = FlagZ, true
		}
		if (c.P&flag != 0) == want {
			off := int8(c.Mem[c.PC])
			c.PC++
			target := c.PC + uint16(int16(off))
			ex = 1
			if c.PC&0xFF00 != target&0xFF00 {
				ex++
			}
			c.PC = target
		} else {
			c.PC++
		}
	case "CLC":
		c.P &^= FlagC
	case "SEC":
		c.P |= FlagC
	case "CLI":
		c.P &^= FlagI
	case "SEI":
		c.P |= FlagI
	case "CLD":
		c.P &^= FlagD
	case "SED":
		c.P |= FlagD
	case "CLV":
		c.P &^= FlagV
	case "PHP":
		c.push(c.P | FlagB | FlagU)
	case "PLP":
		c.P = c.pop() | FlagB | FlagU
	case "PHA":
		c.push(c.A)
	case "PLA":
		c.A = c.pop()
		c.nz(c.A)
	case "DEY":
		c.Y--
		c.nz(c.Y)
	case "INY":
		c.Y++
		c.nz(c.Y)
	case "DEX":
		c.X--
		c.nz(c.X)
	case "INX":
		c.X++
		c.nz(c.X)
	case "TXA":
		c.A = c.X
		c.nz(c.A)
	case "TYA":
		c.A = c.Y
		c.nz(c.A)
	case "TAY":
		c.Y = c.A
		c.nz(c.Y)
	case "TAX":
		c.X = c.A
		c.nz(c.X)
	case "TXS":
		c.SP = c.X
	case "TSX":
		c.X = c.SP
		c.nz(c.X)
	case "NOP":
	case "ASL", "LSR", "ROL", "ROR", "INC", "DEC":
		var a uint16
		var t byte
		if m == acc {
			t = c.A
		} else {
			a = c.addr(m, info.extra, &ex)
			t = c.Mem[a]
		}
		switch info.name {
		case "ASL":
			c.P &^= FlagC | FlagN | FlagZ
			c.P |= t >> 7
			t <<= 1
			c.nz(t)
		case "LSR":
			c.P &^= FlagC | FlagN | FlagZ
			c.P |= t & 1
			t >>= 1
			if t == 0 {
				c.P |= FlagZ
			}
		case "ROL":
			cin := c.P & FlagC
			c.P = c.P&^FlagC | t>>7
			t = t<<1 | cin
			c.nz(t)
		case "ROR":
			cin := c.P & FlagC
			c.P = c.P&^FlagC | t&1
			t = t>>1 | cin<<7
			c.nz(t)
		case "INC":
			t++
			c.nz(t)
		case "DEC":
			t--
			c.nz(t)
		}
		if m == acc {
			c.A = t
		} else {
			c.Mem[a] = t
		}
		c.PC += size(m)
	default:
		a := c.addr(m, info.extra, &ex)
		switch info.name {
		case "STA":
			c.Mem[a] = c.A
		case "STX":
			c.Mem[a] = c.X
		case "STY":
			c.Mem[a] = c.Y
		case "ORA":
			c.A |= c.Mem[a]
			c.nz(c.A)
		case "AND":
			c.A &= c.Mem[a]
			c.nz(c.A)
		case "EOR":
			c.A ^= c.Mem[a]
			c.nz(c.A)
		case "LDA":
			c.A = c.Mem[a]
			c.nz(c.A)
		case "LDX":
			c.X = c.Mem[a]
			c.nz(c.X)
		case "LDY":
			c.Y = c.Mem[a]
			c.nz(c.Y)
		case "ADC":
			c.adc(c.Mem[a])
		case "SBC":
			c.sbc(c.Mem[a])
		case "CMP":
			c.cmp(c.A, c.Mem[a])
		case "CPX":
			c.cmp(c.X, c.Mem[a])
		case "CPY":
			c.cmp(c.Y, c.Mem[a])
		case "BIT":
			t := c.Mem[a]
			c.P &^= FlagZ | FlagN | FlagV
			if c.A&t == 0 {
				c.P |= FlagZ
			}
			c.P |= t & (FlagN | FlagV)
		}
		c.PC += size(m)
	}
	c.Cycles += int64(info.cycles) + ex
}
