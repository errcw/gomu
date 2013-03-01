package main

type Pixel struct {
	R, G, B uint8
}

type Ppu struct {
	framebuffer []Pixel
}

type PpuResult int

const (
	PpuTick = iota
	PpuVblankNmi
	PpuNewFrame
)

func (ppu *Ppu) Step(cycles int) PpuResult {
	return PpuTick
}

func (ppu *Ppu) Load(addr uint16) uint8 {
	return 0
}

func (ppu *Ppu) Store(addr uint16, val uint8) {
}

type VramMemoryMap struct {
	mapper     Mapper
	nametables [0x800]uint8
	palette    [0x20]uint8
}

func (mem *VramMemoryMap) Load(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return mem.mapper.LoadChr(addr)
	case addr < 0x3f00:
		return mem.nametables[addr&0x7ff]
	case addr < 0x4000:
		return mem.palette[addr&0x1f]
	}
	panic("Invalid VRAM address")
}

func (mem *VramMemoryMap) Store(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		mem.mapper.StoreChr(addr, val)
	case addr < 0x3f00:
		mem.nametables[addr&0x7ff] = val
	case addr < 0x4000:
		// TODO Any more to be done here?
		mem.palette[addr&0x1f] = val
	}
}
