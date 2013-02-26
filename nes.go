package main

import "fmt"

func main() {
	rom, err := LoadRom("testdata/instr_test-v3/official_only.nes")
	if err != nil {
		panic(fmt.Sprintf("Failed to load ROM: %v", err))
	}

	ppu := Ppu{}
	apu := Apu{}
	mem := &MemoryMap{ppu: ppu, apu: apu, mapper: NewMapper(rom)}
	cpu := NewCpu(mem)
	cpu.Reset()

	for {
		cycles := cpu.Step()

		ppuResult := ppu.Step(cycles)
		switch ppuResult {
		case PpuVblankNmi:
			cpu.Nmi()
		case PpuNewFrame:
			// blt
		}

		apu.Step(cycles)
	}
}
