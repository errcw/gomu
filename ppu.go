package main

type Pixel struct {
	R, G, B uint8
}

type Ppu struct {
	ctrl        PpuCtrlReg   // PPUCTRL (0x2000)
	mask        PpuMaskReg   // PPUMASK (0x2001)
	status      PpuStatusReg // PPUSTATUS (0x2002)
	oam         uint8        // OAMDATA (0x2004)
	scroll      PpuScrollReg // PPUSCROLL (0x2005)
	addr        PpuAddrReg   // PPUADDR (0x2006)
	data        uint8        // PPUDATA (0x2007)
	framebuffer []Pixel
}

type PpuCtrlReg uint8
type PpuMaskReg uint8
type PpuStatusReg uint8
type PpuScrollReg uint8
type PpuAddrReg uint8

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
	switch addr & 7 {
	case 0:
		return uint8(ppu.ctrl)
	case 1:
		return uint8(ppu.mask)
	case 2:
		return uint8(ppu.status) // TODO
	case 7:
		return ppu.data // TODO
	case 4:
		panic("OAMDATA not implemented")
	case 3, 5, 6:
		return 0 // OAMADDR, PPUSCROLL, PPUADDR are read-only
	}
	panic("Unexpected PPU load")
}

func (ppu *Ppu) Store(addr uint16, val uint8) {
	switch addr & 7 {
	case 0:
		ppu.ctrl = PpuCtrlReg(val) // TODO
	case 1:
		ppu.mask = PpuMaskReg(val)
	case 3:
		ppu.oam = val
	case 4:
		// write oam data
	case 5:
		ppu.scroll = PpuScrollReg(val) // TODO
	case 6:
		ppu.addr = PpuAddrReg(val) // TODO
	case 7:
		ppu.data = val // TODO
	}
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
