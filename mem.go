package main

// CPU bus memory map
type MemoryMap struct {
	ram    [0x800]uint8 // 2KB of RAM
	cpu    *Cpu
	ppu    *Ppu
	apu    *Apu
	input  *Input
	mapper Mapper
}

func (mem *MemoryMap) Load(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return mem.ram[addr&0x7ff]
	case addr < 0x4000:
		return mem.ppu.Load(addr)
	case addr < 0x4016:
		return mem.apu.Load(addr)
	case addr < 0x4018:
		return mem.input.Load(addr)
	}
	return mem.mapper.LoadPrg(addr)
}

func (mem *MemoryMap) Store(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		mem.ram[addr&0x7ff] = val
	case addr < 0x4000:
		mem.ppu.Store(addr, val)
	case addr == 0x4014:
		dma(val)
	case addr == 0x4016:
		mem.input.Store(addr, val)
	case addr <= 0x4018:
		mem.apu.Store(addr, val)
	default:
		mem.mapper.StorePrg(addr, val)
	}
}

func dma(addrHigh uint8) {
}
