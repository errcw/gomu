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

	// Pending interrupt
	interrupt int

	// Behavior of the current instruction
	pageCrossed bool
	branchTaken bool

  Memory
}

const (
	CarryFlag    = 1 << 0
	ZeroFlag     = 1 << 1
	IrqFlag      = 1 << 2
	DecimalFlag  = 1 << 3
	BreakFlag    = 1 << 4
	UnusedFlag   = 1 << 4
	OverflowFlag = 1 << 6
	NegativeFlag = 1 << 7
)

const (
	NmiVector   = 0xfffa
	ResetVector = 0xfffc
	IrqVector   = 0xfffe
)

const (
	InterruptNone = iota
	InterruptIrq
	InterruptNmi
	InterruptReset
)

func (cpu *Cpu) Reset() {
	cpu.a = 0
	cpu.x = 0
	cpu.y = 0
	cpu.sp = 0xfd
	cpu.pc = 0x34
	cpu.flags = 0

	cpu.interrupt = InterruptNone
}

func (cpu *Cpu) Step() int {
	opcode := cpu.loadAndIncPc()
	instruction, ok := instructions[opcode]
	if !ok {
		panic(fmt.Sprintf("Unimplemented/illegal instruction %x", opcode))
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

func makeWord(lowb, highb uint8) uint16 {
	return uint16(highb)<<8 | uint16(lowb)
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
	// STA
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
	0x09: {fn: bit, addr: zeroPage, cycles: 3},
	0x05: {fn: bit, addr: absolute, cycles: 4},
	// ADC
	0x69: {fn: adc, addr: immediate, cycles: 2},
	0x65: {fn: adc, addr: zeroPage, cycles: 3},
	0x75: {fn: adc, addr: zeroPageX, cycles: 4},
	0x6d: {fn: adc, addr: absolute, cycles: 4},
	0x7d: {fn: adc, addr: absoluteX, cycles: 4, hasPageCyclePenalty: true},
	0x79: {fn: adc, addr: absoluteY, cycles: 4, hasPageCyclePenalty: true},
	0x61: {fn: adc, addr: indexedIndirect, cycles: 6},
	0x71: {fn: adc, addr: indirectIndexed, cycles: 5, hasPageCyclePenalty: true},
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
func php(cpu *Cpu, addr AddressFn) { push(cpu, cpu.flags | BreakFlag | UnusedFlag) }
func plp(cpu *Cpu, addr AddressFn) { cpu.flags = pop(cpu) }

func and(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.a & cpu.Load(addr(cpu))) }
func eor(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.a ^ cpu.Load(addr(cpu))) }
func ora(cpu *Cpu, addr AddressFn) { cpu.a = cpu.setNZ(cpu.a | cpu.Load(addr(cpu))) }

func bit(cpu *Cpu, addr AddressFn) {
  val := cpu.Load(addr(cpu))
  cpu.setFlag(ZeroFlag, cpu.a & val == 0)
  cpu.setFlag(OverflowFlag, val & OverflowFlag == OverflowFlag)
  cpu.setFlag(NegativeFlag, val & NegativeFlag == NegativeFlag)
}

func adc(cpu *Cpu, addr AddressFn) {
  val := uint16(cpu.Load(addr(cpu)))
  a := uint16(cpu.a)
  carry := uint16(cpu.flags & CarryFlag)
  result := val + a + carry

  cpu.a = uint8(result & 0xff)
  cpu.setNZ(cpu.a)
  cpu.setFlag(CarryFlag, result & 0x100 == 0x100)
  cpu.setFlag(OverflowFlag, result & 0x100 == 0x100)
}

func push(cpu *Cpu, val uint8) {
	cpu.Store(0x100+uint16(cpu.sp), val)
	cpu.sp--
}

func pop(cpu *Cpu) uint8 {
	cpu.sp++
	return cpu.Load(0x100 + uint16(cpu.sp))
}

// Flags
func (cpu *Cpu) setFlag(flag uint8, on bool) {
	if on {
		cpu.flags |= flag
	} else {
		cpu.flags &= ^flag
	}
}

func (cpu *Cpu) getFlag(flag uint8) bool {
	return (cpu.flags & flag) != 0
}

func (cpu *Cpu) setNZ(val uint8) uint8 {
	cpu.setFlag(NegativeFlag, (val&0x80) != 0)
	cpu.setFlag(ZeroFlag, val == 0)
	return val
}
