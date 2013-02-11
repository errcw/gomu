package main

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
	InterruptReset
	InterruptNmi
)

type Cpu struct {
	// Registers
	a     uint8
	x     uint8
	y     uint8
	sp    uint8
	pc    uint8
	flags uint8

  // Cycle count for the current opcode
	cycles    int
  // Pending interrupt
	interrupt int
}

func (cpu *Cpu) Reset() {
	cpu.a = 0
	cpu.x = 0
	cpu.y = 0
	cpu.sp = 0xfd
	cpu.pc = 0x34
	cpu.flags = 0

	cpu.cycles = 0
	cpu.interrupt = InterruptNone
}

func (cpu *Cpu) Step() int {

	// Load and increment the program counter
	opcode := Mem.LoadB(uint16(cpu.pc))
	cpu.pc += 1

	switch opcode {
  default:
    panic("Illegal or unimplemented opcode")
	}

	return 0
}
