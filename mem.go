package main

type Memory struct {
	ram    [0x800]uint8 // 2KB of RAM
	mapper Mapper
}

func (mem *Memory) Load(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return mem.ram[addr&0x7ff]
	case addr < 0x4000:
		return 0 // PPU (addr & 0x7)
	case addr < 0x4016:
		return 0 // APU
	case addr < 0x4018:
		return 0 // Input (+ more APU, 0x4017 frame counter control?)
	}
	return mem.mapper.Load(addr)
}

func (mem *Memory) Store(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		mem.ram[addr&0x7ff] = val
	case addr < 0x4000:
		// PPU
	case addr < 0x4016:
		// APU
	case addr < 0x4018:
		// Input (+ more APU, 0x4017 frame counter control?)
	default:
		mem.mapper.Store(addr, val)
	}
}
