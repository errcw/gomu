package main

import "testing"

func TestCpuRom(t *testing.T) {
	rom, err := LoadRom("testdata/instr_test-v3/official_only.nes")
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
		return
	}

	nes := NewNes(rom)
	ram := nes.cpu.MemoryMap.mapper.(*Mmc1).prgRam

	for {
		nes.cpu.Step()
		if ram[1] == 0xde && ram[2] == 0xb0 && ram[3] == 0x61 && ram[0] != 0x80 {
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
