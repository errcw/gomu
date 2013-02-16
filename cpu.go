package main

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
}

const (
	CarryFlag    = 1 << 0
	ZeroFlag     = 1 << 1
	IrqFlag      = 1 << 2
	DecimalFlag  = 1 << 3
	BreakFlag    = 1 << 4
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
		panic("Unimplemented/illegal instruction")
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
	val := Mem.Load(cpu.pc)
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
	lowByte := Mem.Load(makeWord(lowByteInd, highByteInd))
	highByte := Mem.Load(makeWord(lowByteInd+1, highByteInd))
	return makeWord(lowByte, highByte)
}

func indexedIndirect(cpu *Cpu) uint16 {
	addr := cpu.loadAndIncPc() + cpu.x
	lowByte := Mem.Load(uint16(addr))
	highByte := Mem.Load(uint16(addr + 1))
	return makeWord(lowByte, highByte)
}

func indirectIndexed(cpu *Cpu) uint16 {
	zeroPageAddr := cpu.loadAndIncPc()
	lowByte := Mem.Load(uint16(zeroPageAddr))
	highByte := Mem.Load(uint16(zeroPageAddr + 1))
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
	0xa9: {fn: lda, addr: immediate},
	0xa5: {fn: lda, addr: zeroPage},
	0xb5: {fn: lda, addr: zeroPageX},
	0xad: {fn: lda, addr: absolute},
	0xbd: {fn: lda, addr: absoluteX},
	0xb9: {fn: lda, addr: absoluteY},
	0xa1: {fn: lda, addr: indexedIndirect},
	0xb1: {fn: lda, addr: indirectIndexed},
	// LDX
	0xa2: {fn: ldx, addr: immediate},
	0xa6: {fn: ldx, addr: zeroPage},
	0xb6: {fn: ldx, addr: zeroPageY},
	0xae: {fn: ldx, addr: absolute},
	0xbe: {fn: ldx, addr: absoluteY},
	// LDY
	0xa0: {fn: ldy, addr: immediate},
	0xa4: {fn: ldy, addr: zeroPage},
	0xb4: {fn: ldy, addr: zeroPageX},
	0xac: {fn: ldy, addr: absolute},
	0xbc: {fn: ldy, addr: absoluteX},
}

func lda(cpu *Cpu, addr AddressFn) {
	cpu.a = cpu.setNZ(Mem.Load(addr(cpu)))
}

func ldx(cpu *Cpu, addr AddressFn) {
	cpu.x = cpu.setNZ(Mem.Load(addr(cpu)))
}

func ldy(cpu *Cpu, addr AddressFn) {
	cpu.y = cpu.setNZ(Mem.Load(addr(cpu)))
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
