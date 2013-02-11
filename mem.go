package main

type Memory struct {
	// 2KB of RAM
	ram [0x800]uint8
}

var Mem Memory

func (mem *Memory) LoadB(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return mem.ram[addr&0x7ff]
	case addr < 0x4000:
		return 0 // PPU (addr & 0x7)
	case addr < 0x4016:
		return 0 // APU
	case addr < 0x4018:
		return 0 // Input (+ more APU, 0x4017 frame counter control?)
	default:
		return 0 // Mapper
	}
}

func (mem *Memory) StoreB(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		mem.ram[addr&0x7ff] = val
	case addr < 0x4000:
		// PPU
	case addr < 0x4016:
		return 0 // APU
	case addr < 0x4018:
		return 0 // Input (+ more APU, 0x4017 frame counter control?)
	default:
		// Mapper
	}
}
