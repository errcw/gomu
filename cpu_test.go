package main

import "testing"

func TestCpuRoms(t *testing.T) {
	//rom, err := LoadRom("testdata/instr_test-v3/rom_singles/02-immediate.nes")
	rom, err := LoadRom("testdata/instr_test-v3/rom_singles/01-implied.nes")
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

  end := 4
  for ; ram[end] != 0; end++ {
  }
  t.Errorf("Ret: %v", ram[0])
  t.Errorf("Test output:%s", string(ram[4:end]))
}
