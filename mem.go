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
		dma(mem, val)
	case addr == 0x4016:
		mem.input.Store(addr, val)
	case addr <= 0x4018:
		mem.apu.Store(addr, val)
	default:
		mem.mapper.StorePrg(addr, val)
	}
}

func dma(mem *MemoryMap, addrHigh uint8) {
	for addrLow := 0; addrLow <= 0xff; addrLow++ {
		addr := uint16(addrHigh)<<8 | uint16(addrLow)
		mem.Store(0x2004, mem.Load(addr))
	}
	// FIXME: Not entirely cycle accurate--starting OAM DMA on CPU read beat (odd
	// cycle) adds an extra cycle to the CPU idle time. Moreover there are
	// complications with APU DMC DMA and cycle stealing.
	mem.cpu.Idle(513)
}
