package main

import "testing"

func TestCpuRoms(t *testing.T) {
	//rom, err := LoadRom("testdata/instr_test-v3/rom_singles/02-immediate.nes")
	rom, err := LoadRom("testdata/instr_test-v3/rom_singles/03-zero_page.nes")
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
		return
	}

	cpu := NewCpu()
	cpu.Memory.mapper = NewMapper(rom)
	cpu.Reset()

	ram := cpu.Memory.mapper.(*Nrom).prgRam

	for {
		cpu.Step()
		if ram[1] == 0xde && ram[2] == 0xb0 && ram[3] == 0x61 && ram[0] < 0x80 {
			break
		}
	}

	returnCode := ram[0]
	if returnCode > 0 {
		t.Errorf("Return: %v", returnCode)
	}

	end := 4
	for ; ram[end] != 0; end++ {
	}
	t.Logf("Test output: %s", string(ram[4:end]))
}
