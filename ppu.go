package main

type Pixel struct {
	R, G, B uint8
}

type Ppu struct {
	ctrl    PpuCtrlReg   // PPUCTRL (0x2000)
	mask    PpuMaskReg   // PPUMASK (0x2001)
	status  PpuStatusReg // PPUSTATUS (0x2002)
	oamAddr uint16       // OAMADDR (0x2003)
	data    uint8        // PPUDATA (0x2007)

	framebuffer []Pixel

	vram *VramMemoryMap
	oam  [0x100]uint8

	vramLatch  uint16
	vramAddr   uint16
	writeLatch bool

	readBuffer uint8

	scrollX uint8

	cycle int
}

type PpuCtrlReg uint8
type PpuMaskReg uint8
type PpuStatusReg uint8

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
	case 2:
		return ppu.readStatus()
	case 4:
		return ppu.readOamData()
	case 7:
		return ppu.readData()
	}
	// Better emulation would simulate the bus hold-up between the PPU and CPU
	// that causes the last value written to be readable for ~600ms
	return 0
}

func (ppu *Ppu) Store(addr uint16, val uint8) {
	switch addr & 7 {
	case 0:
		ppu.writeCtrl(val)
	case 1:
		ppu.writeMask(val)
	case 3:
		ppu.writeOamAddr(val)
	case 4:
		ppu.writeOamData(val)
	case 5:
		ppu.writeScroll(val)
	case 6:
		ppu.writeAddr(val)
	case 7:
		ppu.writeData(val)
	}
}

func (ppu *Ppu) readStatus() uint8 {
	ppu.writeLatch = false

	// Better emulation would drop the vblank flag and suppress NMIs when the
	// status is read at the exact start of vblank
	status := uint8(ppu.status)
	ppu.status.clearVblank()

	return status
}

func (ppu *Ppu) readOamData() uint8 {
	return ppu.oam[ppu.oamAddr]
}

func (ppu *Ppu) readData() uint8 {
	data := ppu.readBuffer
	if ppu.vramAddr < 0x3f00 {
		ppu.readBuffer = ppu.vram.Load(ppu.vramAddr)
	} else {
		data = ppu.vram.Load(ppu.vramAddr)
		ppu.readBuffer = ppu.vram.Load(ppu.vramAddr - 0x1000)
	}
	ppu.vramAddr += ppu.ctrl.vramAddrInc()
	return data
}

func (ppu *Ppu) writeCtrl(val uint8) {
	ppu.ctrl = PpuCtrlReg(val)
	ppu.vramLatch = (ppu.vramLatch & 0xf3ff) | ((uint16(val) & 3) << 10)
}

func (ppu *Ppu) writeMask(val uint8) {
	ppu.mask = PpuMaskReg(val)
}

func (ppu *Ppu) writeOamAddr(val uint8) {
	ppu.oamAddr = uint16(val)
}

func (ppu *Ppu) writeOamData(val uint8) {
	if ppu.oamAddr&3 == 2 { // OAM is only 29 bits, mask off part of byte 2
		val &= 0xe3
	}
	ppu.oam[ppu.oamAddr] = val
	ppu.oamAddr++
}

func (ppu *Ppu) writeScroll(val uint8) {
	if !ppu.writeLatch {
		ppu.vramLatch = (ppu.vramLatch & 0xffe0) | ((uint16(val) & 0xf8) >> 3)
		ppu.scrollX = val & 0x7
	} else {
		ppu.vramLatch = (ppu.vramLatch & 0x8fff) | ((uint16(val) & 0x07) << 12)
		ppu.vramLatch = (ppu.vramLatch & 0xfc1f) | ((uint16(val) & 0xf8) << 2)
	}
	ppu.writeLatch = !ppu.writeLatch
}

func (ppu *Ppu) writeAddr(val uint8) {
	if !ppu.writeLatch {
		ppu.vramLatch = (ppu.vramLatch & 0x00ff) | ((uint16(val) & 0x3f) << 8)
	} else {
		ppu.vramLatch = (ppu.vramLatch & 0xff00) | uint16(val)
		ppu.vramAddr = ppu.vramLatch
	}
	ppu.writeLatch = !ppu.writeLatch
}

func (ppu *Ppu) writeData(val uint8) {
	ppu.vram.Store(ppu.vramAddr, val)
	ppu.vramAddr += ppu.ctrl.vramAddrInc()
}

type VramMemoryMap struct {
	mapper     Mapper
	nametables [2]*[0x400]uint8
	palette    [0x20]uint8
}

const (
	MirrorVertical = iota
	MirrorHorizontal
	MirrorSingleUpper
	MirrorSingleLower
)

type Mirroring int

// Maps logical nametables to physical nametables based on the mirroring configuration
var nametableMirroring = map[Mirroring][4]int{
	MirrorVertical:    {0, 0, 1, 1},
	MirrorHorizontal:  {0, 1, 0, 1},
	MirrorSingleUpper: {0, 0, 0, 0},
	MirrorSingleLower: {1, 1, 1, 1},
}

func (mem *VramMemoryMap) Load(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return mem.mapper.LoadChr(addr)
	case addr < 0x3f00:
		nametable := nametableMirroring[mem.mapper.Mirroring()][(addr&0xc00)>>10]
		return mem.nametables[nametable][addr&0x3ff]
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
		nametable := nametableMirroring[mem.mapper.Mirroring()][(addr&0xc00)>>10]
		mem.nametables[nametable][addr&0x3ff] = val
	case addr < 0x4000:
		if addr&0xf == 0 {
			mem.palette[0x00] = val
			mem.palette[0x10] = val
		} else {
			mem.palette[addr&0x1f] = val
		}
	}
}

func (ctrl PpuCtrlReg) vramAddrInc() uint16 {
	if ctrl&0x4 == 1 {
		return 32
	}
	return 1
}

func (status *PpuStatusReg) setSpriteOverflow()   { *status |= 0x20 }
func (status *PpuStatusReg) clearSpriteOverflow() { *status &= 0xdf }
func (status *PpuStatusReg) setSprite0Hit()       { *status |= 0x40 }
func (status *PpuStatusReg) clearSprite0Hit()     { *status &= 0xbf }
func (status *PpuStatusReg) setVblank()           { *status |= 0x80 }
func (status *PpuStatusReg) clearVblank()         { *status &= 0x7f }
