package main

import "testing"

func TestCpuWithSimpleInstructions(t *testing.T) {
	cpu := new(Cpu)
	cpu.Reset()

	// Zero page data
	cpu.ram[0x11] = 0xee

	// LDX #11 (X = 0x1)
	cpu.ram[0x34] = 0xa2
	cpu.ram[0x35] = 0x1

	// LDA $11 (A = mem(0x11) = 0xee)
	cpu.ram[0x36] = 0xb5
	cpu.ram[0x37] = 0x10

	// STA $1110 (mem(0x620) = A = 0xee)
	cpu.ram[0x38] = 0x8d
	cpu.ram[0x39] = 0x20
	cpu.ram[0x3a] = 0x06

	for i := 0; i < 3; i++ {
		cpu.Step()
	}

	if cpu.pc != 0x3b {
		t.Errorf("PC %x != 0x3b", cpu.pc)
	}
	if cpu.ram[0x620] != 0xee {
		t.Errorf("RAM[0x620] %x != 0xee", cpu.ram[0x620])
	}
}

func TestAddOverflow(t *testing.T) {
	for i := 0; i <= 0xff; i++ {
		for j := 0; j <= 0xff; j++ {
      // Check if the signed result overflowed
			sr := int16(int8(i)) + int16(int8(j))
			v := sr < -128 || sr > 127

      // Check if the unsigned algorithm works
			ii := uint8(i)
			jj := uint8(j)
			r := ii + jj
			u := ((ii^jj)&0x80 == 0x0) && ((ii^r)&0x80 == 0x80)

			if v != u {
				t.Errorf("%x + %x = %x (%v + %v = %v), %v != %v", ii, jj, r, int8(i), int8(j), sr, v, u)
			}
		}
	}
}
