package main

import "fmt"

type Cpu struct {
	// Registers
	a     uint8
	x     uint8
	y     uint8
	sp    uint8
	pc    uint16
	flags uint8

	// Behavior of the current instruction
	pageCrossed bool
	branchTaken bool

	// Memory map
	Memory

  verbose bool
  count int
}

const (
	CarryFlag    = 1 << 0
	ZeroFlag     = 1 << 1
	IrqFlag      = 1 << 2
	DecimalFlag  = 1 << 3
	BreakFlag    = 1 << 4
	UnusedFlag   = 1 << 5
	OverflowFlag = 1 << 6
	NegativeFlag = 1 << 7
)

const (
	NmiVector   = 0xfffa
	ResetVector = 0xfffc
	IrqVector   = 0xfffe
)

func NewCpu() *Cpu {
	return &Cpu{a: 0, x: 0, y: 0, sp: 0xfd, pc: 0xc000, flags: IrqFlag | UnusedFlag}
}

func (cpu *Cpu) Reset() {
	lowByte := cpu.Load(ResetVector)
	highByte := cpu.Load(ResetVector + 1)
	cpu.pc = makeWord(lowByte, highByte)
}

func (cpu *Cpu) Step() int {
	opcode := cpu.loadAndIncPc()
	instruction, ok := instructions[opcode]
  if opcode == 0x60 {
    cpu.verbose = true
  }
  if cpu.verbose {
    fmt.Printf("Executing %x at %x\n", opcode, cpu.pc - 1)
    cpu.count++
  }
	if !ok {
		panic(fmt.Sprintf("Unimplemented/illegal instruction %x at %x", opcode, cpu.pc-1))
	}
	instruction.fn(cpu, instruction.addr)

	cycles := instruction.cycles
	if cpu.pageCrossed && instruction.hasPageCyclePenalty {
		cycles++
		cpu.pageCrossed = false
	}
	if cpu.branchTaken && instruction.hasBranchCyclePenalty {
		cycles++
		cpu.branchTaken = false
	}
	return cycles
}

func (cpu *Cpu) loadAndIncPc() uint8 {
	val := cpu.Load(cpu.pc)
	cpu.pc++
	return val
}

// Flags
func (cpu *Cpu) setFlag(flag uint8, on bool) {
	if on {
		cpu.flags |= flag
	} else {
		cpu.flags &^= flag
	}
}

func (cpu *Cpu) setNZ(val uint8) uint8 {
	cpu.setFlag(NegativeFlag, (val&0x80) != 0)
	cpu.setFlag(ZeroFlag, val == 0)
	return val
}

// Addressing modes
func immediate(cpu *Cpu) uint16 {
	cpu.pc++
	return cpu.pc - 1
}

func zeroPage(cpu *Cpu) uint16 {
	return uint16(cpu.loadAndIncPc())
}

func zeroPageX(cpu *Cpu) uint16 {
	return uint16(uint8(zeroPage(cpu)) + cpu.x)
}

func zeroPageY(cpu *Cpu) uint16 {
	return uint16(uint8(zeroPage(cpu)) + cpu.y)
}

func absolute(cpu *Cpu) uint16 {
	lowByte := cpu.loadAndIncPc()
	highByte := cpu.loadAndIncPc()
	return makeWord(lowByte, highByte)
}

func absoluteX(cpu *Cpu) uint16 {
	return indexed(cpu, absolute(cpu), cpu.x)
}

func absoluteY(cpu *Cpu) uint16 {
	return indexed(cpu, absolute(cpu), cpu.y)
}

func indirect(cpu *Cpu) uint16 {
	lowByteInd := cpu.loadAndIncPc()
	highByteInd := cpu.loadAndIncPc()
	lowByte := cpu.Load(makeWord(lowByteInd, highByteInd))
	highByte := cpu.Load(makeWord(lowByteInd+1, highByteInd))
	return makeWord(lowByte, highByte)
}

func indexedIndirect(cpu *Cpu) uint16 {
	addr := cpu.loadAndIncPc() + cpu.x
	lowByte := cpu.Load(uint16(addr))
	highByte := cpu.Load(uint16(addr + 1))
	return makeWord(lowByte, highByte)
}

func indirectIndexed(cpu *Cpu) uint16 {
	zeroPageAddr := cpu.loadAndIncPc()
	lowByte := cpu.Load(uint16(zeroPageAddr))
	highByte := cpu.Load(uint16(zeroPageAddr + 1))
	return indexed(cpu, makeWord(lowByte, highByte), cpu.y)
}

func relative(cpu *Cpu) uint16 {
	base := cpu.pc
	raw := cpu.loadAndIncPc()
	offset := int8(raw)
	addr := uint16(int16(cpu.pc) + int16(offset))
	if base&0xff00 != addr&0xff00 {
		cpu.pageCrossed = true
	}
	return addr
}

func implied(cpu *Cpu) uint16 {
	panic("Implied addressing should never be invoked")
}

func indexed(cpu *Cpu, base uint16, index uint8) uint16 {
	indexed := base + uint16(index)
	if base&0xff00 != indexed&0xff00 {
		cpu.pageCrossed = true
	}
	return indexed
}

// Instructions
type InstructionFn func(*Cpu, AddressFn)
type AddressFn func(*Cpu) uint16

type Instruction struct {
	fn   InstructionFn
	addr AddressFn

	// Number of cycles taken by this instruction, including extra cycles if the as
	// address crosses a page boundary or a branch is taken
	cycles                int
	hasPageCyclePenalty   bool
	hasBranchCyclePenalty bool
}

var instructions = map[uint8]Instruction{
	// LDA
	0xa9: {fn: lda, addr: immediate, cycles: 2},
	0xa5: {fn: lda, addr: zeroPage, cycles: 3},
	0xb5: {fn: lda, addr: zeroPageX, cycles: 4},
	0xad: {fn: lda, addr: absolute, cycles: 4},
	0xbd: {fn: lda, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0xb9: {fn: lda, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0xa1: {fn: lda, addr: indexedIndirect, cycles: 6},
	0xb1: {fn: lda, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
	// LDX
	0xa2: {fn: ldx, addr: immediate, cycles: 2},
	0xa6: {fn: ldx, addr: zeroPage, cycles: 3},
	0xb6: {fn: ldx, addr: zeroPageY, cycles: 4},
	0xae: {fn: ldx, addr: absolute, cycles: 4},
	0xbe: {fn: ldx, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	// LDY
	0xa0: {fn: ldy, addr: immediate, cycles: 2},
	0xa4: {fn: ldy, addr: zeroPage, cycles: 3},
	0xb4: {fn: ldy, addr: zeroPageX, cycles: 4},
	0xac: {fn: ldy, addr: absolute, cycles: 4},
	0xbc: {fn: ldy, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	// STX
	0x85: {fn: sta, addr: zeroPage, cycles: 3},
	0x95: {fn: sta, addr: zeroPageX, cycles: 4},
	0x8d: {fn: sta, addr: absolute, cycles: 4},
	0x9d: {fn: sta, addr: absoluteX, cycles: 5},
	0x99: {fn: sta, addr: absoluteY, cycles: 5},
	0x81: {fn: sta, addr: indexedIndirect, cycles: 6},
	0x91: {fn: sta, addr: indirectIndexed, cycles: 6},
	// STX
	0x86: {fn: stx, addr: zeroPage, cycles: 3},
	0x96: {fn: stx, addr: zeroPageY, cycles: 4},
	0x8e: {fn: stx, addr: absolute, cycles: 4},
	// STY
	0x84: {fn: sty, addr: zeroPage, cycles: 3},
	0x94: {fn: sty, addr: zeroPageX, cycles: 4},
	0x8c: {fn: sty, addr: absolute, cycles: 4},
	// TAX, TAY, TXA, TYA, TSX, TXS
	0xaa: {fn: tax, addr: implied, cycles: 2},
	0xa8: {fn: tay, addr: implied, cycles: 2},
	0x8a: {fn: txa, addr: implied, cycles: 2},
	0x98: {fn: tya, addr: implied, cycles: 2},
	0xba: {fn: tsx, addr: implied, cycles: 2},
	0x9a: {fn: txs, addr: implied, cycles: 2},
	// PHA, PLA, PHP, PLP
	0x48: {fn: pha, addr: implied, cycles: 3},
	0x68: {fn: pla, addr: implied, cycles: 4},
	0x08: {fn: php, addr: implied, cycles: 3},
	0x28: {fn: plp, addr: implied, cycles: 4},
	// AND
	0x29: {fn: and, addr: immediate, cycles: 2},
	0x25: {fn: and, addr: zeroPage, cycles: 3},
	0x35: {fn: and, addr: zeroPageX, cycles: 4},
	0x2d: {fn: and, addr: absolute, cycles: 4},
	0x3d: {fn: and, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0x39: {fn: and, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0x21: {fn: and, addr: indexedIndirect, cycles: 6},
	0x31: {fn: and, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
	// EOR
	0x49: {fn: eor, addr: immediate, cycles: 2},
	0x45: {fn: eor, addr: zeroPage, cycles: 3},
	0x55: {fn: eor, addr: zeroPageX, cycles: 4},
	0x4d: {fn: eor, addr: absolute, cycles: 4},
	0x5d: {fn: eor, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0x59: {fn: eor, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0x41: {fn: eor, addr: indexedIndirect, cycles: 6},
	0x51: {fn: eor, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
	// ORA
	0x09: {fn: ora, addr: immediate, cycles: 2},
	0x05: {fn: ora, addr: zeroPage, cycles: 3},
	0x15: {fn: ora, addr: zeroPageX, cycles: 4},
	0x0d: {fn: ora, addr: absolute, cycles: 4},
	0x1d: {fn: ora, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0x19: {fn: ora, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0x01: {fn: ora, addr: indexedIndirect, cycles: 6},
	0x11: {fn: ora, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
	// BIT
	0x24: {fn: bit, addr: zeroPage, cycles: 3},
	0x2c: {fn: bit, addr: absolute, cycles: 4},
	// ADC
	0x69: {fn: adc, addr: immediate, cycles: 2},
	0x65: {fn: adc, addr: zeroPage, cycles: 3},
	0x75: {fn: adc, addr: zeroPageX, cycles: 4},
	0x6d: {fn: adc, addr: absolute, cycles: 4},
	0x7d: {fn: adc, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0x79: {fn: adc, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0x61: {fn: adc, addr: indexedIndirect, cycles: 6},
	0x71: {fn: adc, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
	// SBC
	0xe9: {fn: sbc, addr: immediate, cycles: 2},
	0xe5: {fn: sbc, addr: zeroPage, cycles: 3},
	0xf5: {fn: sbc, addr: zeroPageX, cycles: 4},
	0xed: {fn: sbc, addr: absolute, cycles: 4},
	0xfd: {fn: sbc, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0xf9: {fn: sbc, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0xe1: {fn: sbc, addr: indexedIndirect, cycles: 6},
	0xf1: {fn: sbc, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
	// CMP
	0xc9: {fn: cmp, addr: immediate, cycles: 2},
	0xc5: {fn: cmp, addr: zeroPage, cycles: 3},
	0xd5: {fn: cmp, addr: zeroPageX, cycles: 4},
	0xcd: {fn: cmp, addr: absolute, cycles: 4},
	0xdd: {fn: cmp, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0xd9: {fn: cmp, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0xc1: {fn: cmp, addr: indexedIndirect, cycles: 6},
	0xd1: {fn: cmp, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
	// CPX
	0xe0: {fn: cpx, addr: immediate, cycles: 2},
	0xe4: {fn: cpx, addr: zeroPage, cycles: 3},
	0xec: {fn: cpx, addr: absolute, cycles: 4},
	// CPY
	0xc0: {fn: cpy, addr: immediate, cycles: 2},
	0xc4: {fn: cpy, addr: zeroPage, cycles: 3},
	0xcc: {fn: cpy, addr: absolute, cycles: 4},
	// INC
	0xe6: {fn: inc, addr: zeroPage, cycles: 5},
	0xf6: {fn: inc, addr: zeroPageX, cycles: 6},
	0xee: {fn: inc, addr: absolute, cycles: 6},
	0xfe: {fn: inc, addr: absoluteX, cycles: 7},
	// INX, INY
	0xe8: {fn: inx, addr: implied, cycles: 2},
	0xc8: {fn: iny, addr: implied, cycles: 2},
	// DEC
	0xc6: {fn: dec, addr: zeroPage, cycles: 5},
	0xd6: {fn: dec, addr: zeroPageX, cycles: 6},
	0xce: {fn: dec, addr: absolute, cycles: 6},
	0xde: {fn: dec, addr: absoluteX, cycles: 7},
	// DEX, DEY
	0xca: {fn: dex, addr: implied, cycles: 2},
	0x88: {fn: dey, addr: implied, cycles: 2},
	// ASL
	0x0a: {fn: asla, addr: implied, cycles: 2},
	0x06: {fn: asl, addr: zeroPage, cycles: 5},
	0x16: {fn: asl, addr: zeroPageX, cycles: 6},
	0x0e: {fn: asl, addr: absolute, cycles: 6},
	0x1e: {fn: asl, addr: absoluteX, cycles: 7},
	// LSR
	0x4a: {fn: lsra, addr: implied, cycles: 2},
	0x46: {fn: lsr, addr: zeroPage, cycles: 5},
	0x56: {fn: lsr, addr: zeroPageX, cycles: 6},
	0x4e: {fn: lsr, addr: absolute, cycles: 6},
	0x5e: {fn: lsr, addr: absoluteX, cycles: 7},
	// ROL
	0x2a: {fn: rola, addr: implied, cycles: 2},
	0x26: {fn: rol, addr: zeroPage, cycles: 5},
	0x36: {fn: rol, addr: zeroPageX, cycles: 6},
	0x2e: {fn: rol, addr: absolute, cycles: 6},
	0x3e: {fn: rol, addr: absoluteX, cycles: 7},
	// ROR
	0x6a: {fn: rora, addr: implied, cycles: 2},
	0x66: {fn: ror, addr: zeroPage, cycles: 5},
	0x76: {fn: ror, addr: zeroPageX, cycles: 6},
	0x6e: {fn: ror, addr: absolute, cycles: 6},
	0x7e: {fn: ror, addr: absoluteX, cycles: 7},
	// JMP
	0x4c: {fn: jmp, addr: absolute, cycles: 3},
	0x6c: {fn: jmp, addr: indirect, cycles: 5},
	// JSR, RTS
	0x20: {fn: jsr, addr: absolute, cycles: 6},
	0x60: {fn: rts, addr: implied, cycles: 6},
	// BCC, BCS, BNE, BEQ, BPL, BMI, BVC, BVS
	0x90: {fn: bcc, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	0xb0: {fn: bcs, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	0xd0: {fn: bne, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	0xf0: {fn: beq, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	0x10: {fn: bpl, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	0x30: {fn: bmi, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	0x50: {fn: bvc, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	0x70: {fn: bvs, addr: relative, cycles: 2, hasPageCyclePenalty: true, hasBranchCyclePenalty: true},
	// CLC, CLD, CLI, CLV, SEC, SED, SEI
	0x18: {fn: clc, addr: implied, cycles: 2},
	0xd8: {fn: cld, addr: implied, cycles: 2},
	0x58: {fn: cli, addr: implied, cycles: 2},
	0xb8: {fn: clv, addr: implied, cycles: 2},
	0x38: {fn: sec, addr: implied, cycles: 2},
	0xf8: {fn: sed, addr: implied, cycles: 2},
	0x78: {fn: sei, addr: implied, cycles: 2},
	// BRK, RTI
	0x00: {fn: brk, addr: implied, cycles: 7},
	0x40: {fn: rti, addr: implied, cycles: 6},
	// NOP
	0xea: {fn: nop, addr: implied, cycles: 2},
}

func lda(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.Load(addr(cpu))) }
func ldx(cpu *Cpu, addr AddressFn) { cpu.x = cpu.setNZ(cpu.Load(addr(cpu))) }
func ldy(cpu *Cpu, addr AddressFn) { cpu.y = cpu.setNZ(cpu.Load(addr(cpu))) }

func sta(cpu *Cpu, addr AddressFn) { cpu.Store(addr(cpu), cpu.a) }
func stx(cpu *Cpu, addr AddressFn) { cpu.Store(addr(cpu), cpu.x) }
func sty(cpu *Cpu, addr AddressFn) { cpu.Store(addr(cpu), cpu.y) }

func tax(cpu *Cpu, addr AddressFn) { cpu.x = cpu.setNZ(cpu.a) }
func tay(cpu *Cpu, addr AddressFn) { cpu.y = cpu.setNZ(cpu.a) }
func txa(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.x) }
func tya(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.y) }
func tsx(cpu *Cpu, addr AddressFn) { cpu.x = cpu.setNZ(cpu.sp) }
func txs(cpu *Cpu, addr AddressFn) { cpu.sp = cpu.x }

func pha(cpu *Cpu, addr AddressFn) { push(cpu, cpu.a) }
func pla(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(pop(cpu)) }
func php(cpu *Cpu, addr AddressFn) { push(cpu, cpu.flags|BreakFlag|UnusedFlag) }
func plp(cpu *Cpu, addr AddressFn) { cpu.flags = pop(cpu) }

func and(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.a & cpu.Load(addr(cpu))) }
func eor(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.a ^ cpu.Load(addr(cpu))) }
func ora(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.a | cpu.Load(addr(cpu))) }

func bit(cpu *Cpu, addr AddressFn) {
	val := cpu.Load(addr(cpu))
	cpu.setFlag(ZeroFlag, cpu.a&val == 0)
	cpu.setFlag(OverflowFlag, val&OverflowFlag == OverflowFlag)
	cpu.setFlag(NegativeFlag, val&NegativeFlag == NegativeFlag)
}

func adc(cpu *Cpu, addr AddressFn) {
	v := uint16(cpu.Load(addr(cpu)))
	a := uint16(cpu.a)
	c := uint16(cpu.flags & CarryFlag)
	r := a + v + c

	cpu.a = uint8(r & 0xff)
	cpu.setNZ(cpu.a)
	cpu.setFlag(CarryFlag, r&0x100 == 0x100)
	cpu.setFlag(OverflowFlag, ((a^v)&0x80 == 0) && ((a^r)&0x80 == 0x80)) // Same sign in, different sign out
}

func sbc(cpu *Cpu, addr AddressFn) {
	v := cpu.Load(addr(cpu))
	a := cpu.a
	c := cpu.flags & CarryFlag
	r := a - v - (1 - c)

	cpu.a = r
	cpu.setNZ(cpu.a)
	cpu.setFlag(CarryFlag, r&0x80 == 0)                                  // Borrow (1-c) set when result negative
	cpu.setFlag(OverflowFlag, ((a^v)&0x80 == 0x80) && ((v^r)&0x80 == 0)) // Diff sign in, same sign out
}

func cmp(cpu *Cpu, addr AddressFn) { compare(cpu, cpu.a, cpu.Load(addr(cpu))) }
func cpx(cpu *Cpu, addr AddressFn) { compare(cpu, cpu.x, cpu.Load(addr(cpu))) }
func cpy(cpu *Cpu, addr AddressFn) { compare(cpu, cpu.y, cpu.Load(addr(cpu))) }

func inc(cpu *Cpu, addr AddressFn) {
	a := addr(cpu)
	v := cpu.Load(a) + 1
	cpu.Store(a, cpu.setNZ(v))
}

func inx(cpu *Cpu, addr AddressFn) { cpu.x = cpu.setNZ(cpu.x + 1) }
func iny(cpu *Cpu, addr AddressFn) { cpu.y = cpu.setNZ(cpu.y + 1) }

func dec(cpu *Cpu, addr AddressFn) {
	a := addr(cpu)
	v := cpu.Load(a) - 1
	cpu.Store(a, cpu.setNZ(v))
}

func dex(cpu *Cpu, addr AddressFn) { cpu.x = cpu.setNZ(cpu.x - 1) }
func dey(cpu *Cpu, addr AddressFn) { cpu.y = cpu.setNZ(cpu.y - 1) }

func asla(cpu *Cpu, addr AddressFn) {
	cpu.setFlag(CarryFlag, cpu.a&0x80 == 0x80)
	cpu.a = cpu.setNZ(cpu.a << 1)
}

func asl(cpu *Cpu, addr AddressFn) {
	a := addr(cpu)
	v := cpu.Load(a)
	cpu.setFlag(CarryFlag, v&0x80 == 0x80)
	cpu.Store(a, cpu.setNZ(v<<1))
}

func lsra(cpu *Cpu, addr AddressFn) {
	cpu.setFlag(CarryFlag, cpu.a&1 == 1)
	cpu.a = cpu.setNZ(cpu.a >> 1)
}

func lsr(cpu *Cpu, addr AddressFn) {
	a := addr(cpu)
	v := cpu.Load(a)
	cpu.setFlag(CarryFlag, v&1 == 1)
	cpu.Store(a, cpu.setNZ(v>>1))
}

func rola(cpu *Cpu, addr AddressFn) {
	carry := cpu.flags & CarryFlag
	cpu.setFlag(CarryFlag, cpu.a&0x80 == 0x80)
	cpu.a = cpu.setNZ((cpu.a << 1) | carry)
}

func rol(cpu *Cpu, addr AddressFn) {
	a := addr(cpu)
	v := cpu.Load(a)
	carry := cpu.flags & CarryFlag
	cpu.setFlag(CarryFlag, v&0x80 == 0x80)
	cpu.Store(a, cpu.setNZ((v<<1)|carry))
}

func rora(cpu *Cpu, addr AddressFn) {
	carry := (cpu.flags & CarryFlag) << 8
	cpu.setFlag(CarryFlag, cpu.a&1 == 1)
	cpu.a = cpu.setNZ((cpu.a >> 1) | carry)
}

func ror(cpu *Cpu, addr AddressFn) {
	a := addr(cpu)
	v := cpu.Load(a)
	carry := (cpu.flags & CarryFlag) << 8
	cpu.setFlag(CarryFlag, v&1 == 1)
	cpu.Store(a, cpu.setNZ((v>>1)|carry))
}

func jmp(cpu *Cpu, addr AddressFn) {
	cpu.pc = addr(cpu)
}

func jsr(cpu *Cpu, addr AddressFn) {
  jmpAddr := addr(cpu) // Read the addr bytes first to move the PC
	ret := cpu.pc - 1
	push(cpu, uint8(ret>>8))
	push(cpu, uint8(ret&0xff))
	cpu.pc = jmpAddr
}

func rts(cpu *Cpu, addr AddressFn) {
	lowByte := pop(cpu)
	highByte := pop(cpu)
	cpu.pc = makeWord(lowByte, highByte) + 1
}

func bcc(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&CarryFlag) == 0) }
func bcs(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&CarryFlag) == CarryFlag) }
func bne(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&ZeroFlag) == 0) }
func beq(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&ZeroFlag) == ZeroFlag) }
func bpl(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&NegativeFlag) == 0) }
func bmi(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&NegativeFlag) == NegativeFlag) }
func bvc(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&OverflowFlag) == 0) }
func bvs(cpu *Cpu, addr AddressFn) { branch(cpu, addr, (cpu.flags&OverflowFlag) == OverflowFlag) }

func clc(cpu *Cpu, addr AddressFn) { cpu.setFlag(CarryFlag, false) }
func cld(cpu *Cpu, addr AddressFn) { cpu.setFlag(DecimalFlag, false) }
func cli(cpu *Cpu, addr AddressFn) { cpu.setFlag(IrqFlag, false) }
func clv(cpu *Cpu, addr AddressFn) { cpu.setFlag(OverflowFlag, false) }
func sec(cpu *Cpu, addr AddressFn) { cpu.setFlag(CarryFlag, true) }
func sed(cpu *Cpu, addr AddressFn) { cpu.setFlag(DecimalFlag, true) }
func sei(cpu *Cpu, addr AddressFn) { cpu.setFlag(IrqFlag, true) }

func brk(cpu *Cpu, addr AddressFn) {
	push(cpu, uint8(cpu.pc>>8))
	push(cpu, uint8(cpu.pc&0xff))
	push(cpu, cpu.flags|BreakFlag|UnusedFlag)

	cpu.setFlag(IrqFlag, true)

	lowIrqAddr := cpu.Load(IrqVector)
	highIrqAddr := cpu.Load(IrqVector + 1)
	cpu.pc = makeWord(lowIrqAddr, highIrqAddr) + 1
}

func rti(cpu *Cpu, addr AddressFn) {
	cpu.flags = pop(cpu)
	lowPcAddr := pop(cpu)
	highPcAddr := pop(cpu)
	cpu.pc = makeWord(lowPcAddr, highPcAddr)
}

func nop(cpu *Cpu, addr AddressFn) {}

// Helpers
func push(cpu *Cpu, val uint8) {
	cpu.Store(0x100+uint16(cpu.sp), val)
	cpu.sp--
}

func pop(cpu *Cpu) uint8 {
	cpu.sp++
	return cpu.Load(0x100 + uint16(cpu.sp))
}

func compare(cpu *Cpu, reg uint8, val uint8) {
	cpu.setFlag(CarryFlag, reg >= val)
	cpu.setFlag(ZeroFlag, reg == val)
	cpu.setFlag(NegativeFlag, reg < val)
}

func branch(cpu *Cpu, addr AddressFn, cond bool) {
  a := addr(cpu) // Always read the address to move the PC
	if cond {
    cpu.pc = a
		cpu.branchTaken = true
	}
}

func makeWord(lowb, highb uint8) uint16 {
	return uint16(highb)<<8 | uint16(lowb)
}
