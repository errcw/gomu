package main

type Ppu struct {
	ctrl   PpuCtrlReg   // PPUCTRL (0x2000)
	mask   PpuMaskReg   // PPUMASK (0x2001)
	status PpuStatusReg // PPUSTATUS (0x2002)

	oamAddr uint16 // OAMADDR (0x2003)
	data    uint8  // PPUDATA (0x2007)

	vramLatch  uint16
	vramAddr   uint16
	writeLatch bool
	readBuffer uint8

  pbuffer     []PpuPixel
	Framebuffer []Pixel

	vram *VramMemoryMap
	oam  [0x100]uint8

	scrollX uint8

	cycle    int
	scanline int
	frame    int
}

type PpuCtrlReg uint8
type PpuMaskReg uint8
type PpuStatusReg uint8

const PpuPrerenderScanline = -1
const PpuQuietScanline = 240
const PpuVblankStartScanline = 241
const PpuVblankEndScanline = 260
const PpuCyclesPerScanline = 341

var PaletteRgb = []uint32{
  0x666666, 0x002A88, 0x1412A7, 0x3B00A4, 0x5C007E,
  0x6E0040, 0x6C0600, 0x561D00, 0x333500, 0x0B4800,
  0x005200, 0x004F08, 0x00404D, 0x000000, 0x000000,
  0x000000, 0xADADAD, 0x155FD9, 0x4240FF, 0x7527FE,
  0xA01ACC, 0xB71E7B, 0xB53120, 0x994E00, 0x6B6D00,
  0x388700, 0x0C9300, 0x008F32, 0x007C8D, 0x000000,
  0x000000, 0x000000, 0xFFFEFF, 0x64B0FF, 0x9290FF,
  0xC676FF, 0xF36AFF, 0xFE6ECC, 0xFE8170, 0xEA9E22,
  0xBCBE00, 0x88D800, 0x5CE430, 0x45E082, 0x48CDDE,
  0x4F4F4F, 0x000000, 0x000000, 0xFFFEFF, 0xC0DFFF,
  0xD3D2FF, 0xE8C8FF, 0xFBC2FF, 0xFEC4EA, 0xFECCC5,
  0xF7D8A5, 0xE4E594, 0xCFEF96, 0xBDF4AB, 0xB3F3CC,
  0xB5EBF2, 0xB8B8B8, 0x000000, 0x000000,
}


type PpuPixel struct {
  color uint32
  value int
  index int
}

type Pixel struct {
	R, G, B uint8
}

type PpuResult int

const (
	PpuTick = iota
	PpuVblankNmi
	PpuNewFrame
)

func (ppu *Ppu) Setup() {
	ppu.Framebuffer = make([]Pixel, 0xf000)
	ppu.scanline = 241
}

func (ppu *Ppu) Step() PpuResult {
	ret := PpuResult(PpuTick)

	switch {
	case ppu.scanline == PpuPrerenderScanline:
		ppu.prerenderScanlineCycle()
	case ppu.scanline < PpuQuietScanline:
		ppu.renderScanlineCycle()
	case ppu.scanline == PpuQuietScanline:
		// PPU is idle for one scanline before vblank starts
	case ppu.scanline >= PpuVblankStartScanline:
		nmi := ppu.vblankScanlineCycle()
		if nmi {
			ret = PpuVblankNmi
		}
	}

	if ppu.cycle == PpuCyclesPerScanline {
		ppu.cycle = 0
		ppu.scanline++
	}

	if ppu.scanline > PpuVblankEndScanline {
		ppu.scanline = PpuPrerenderScanline
		ppu.frame++
    ppu.copyFrame()
		ret = PpuNewFrame
	}

	ppu.cycle++

	return ret
}

func (ppu *Ppu) prerenderScanlineCycle() {
	switch ppu.cycle {
	case 1:
		ppu.status.clearVblank()
		ppu.status.clearSprite0Hit()
		ppu.status.clearSpriteOverflow()
		// Better emulation would set during the right cycle (316 in last vblank scanline?)
		ppu.oamAddr = 0x0
	case 304:
		if ppu.mask.showBackground() || ppu.mask.showSprites() {
			ppu.vramAddr = ppu.vramLatch
		}
	}
}

func (ppu *Ppu) renderScanlineCycle() {
	switch ppu.cycle {
	case 254:
		if ppu.mask.showBackground() {
      ppu.renderBackground()
		}
		if ppu.mask.showSprites() {
      ppu.renderSprites()
		}
	case 256:
		if ppu.mask.showBackground() || ppu.mask.showSprites() {
			ppu.updateVramAddrForScanline()
		}
	}
}

func (ppu *Ppu) vblankScanlineCycle() bool {
	if ppu.scanline == PpuVblankStartScanline && ppu.cycle == 1 {
		ppu.status.setVblank()
		if ppu.ctrl.vblankNmi() {
			return true
		}
	}
	return false
}

func (ppu *Ppu) updateVramAddrForScanline() {
	// On cycles 323 and 331 (FIXME Put after other updates?)
	ppu.incrementCoarseXScroll()
	ppu.incrementCoarseXScroll()

  // On cycle 256
	if ppu.vramAddr&0x7000 == 0x7000 {
    // Update coarse y scroll, zero fine y
    sw := ppu.vramAddr & 0x3e0
		ppu.vramAddr &= 0xfff
		switch sw {
		case 0x3a0:
			ppu.vramAddr ^= 0xba0
		case 0x3e0:
			ppu.vramAddr ^= 0x3e0
		default:
			ppu.vramAddr += 0x20
		}
	} else {
		// Update fine y scroll
		ppu.vramAddr += 0x1000
	}

  // On cycle 257
	ppu.vramAddr = (ppu.vramAddr & 0x7be0) | (ppu.vramLatch & 0x41f)
}

func (ppu *Ppu) incrementCoarseXScroll() {
	if ppu.vramAddr&0x1f != 0x1f {
		ppu.vramAddr++
	} else {
		ppu.vramAddr ^= 0x41f
	}
}

func (ppu *Ppu) renderBackground() {
}

func (ppu *Ppu) renderSprites() {
}

func (ppu *Ppu) copyFrame() {
  for i, pixel := range ppu.pbuffer {
    // Compute the RGB color
    rgb := PaletteRgb[pixel.color]
    ppu.Framebuffer[i].R = uint8((rgb >> 16) & 0xff)
    ppu.Framebuffer[i].G = uint8((rgb >> 8) & 0xff)
    ppu.Framebuffer[i].B = uint8(rgb & 0xff)

    // Reset the internal pixels for the next frame
    pixel.value = 0
    pixel.index = -1
  }
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
		// For palette reads the buffer is still populated with a value
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
	nametables [2][0x400]uint8
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

func (ctrl PpuCtrlReg) baseNametableAddress() uint16 {
	switch ctrl & 0x3 {
	case 0:
		return 0x2000
	case 1:
		return 0x2400
	case 2:
		return 0x2800
	case 3:
		return 0x2c00
	}
	panic("Invalid control register state")
}
func (ctrl PpuCtrlReg) vramAddrInc() uint16 {
	if (ctrl>>2)&1 == 1 {
		return 32
	}
	return 1
}
func (ctrl PpuCtrlReg) spritePatternAddress() uint16 {
	if (ctrl>>3)&1 == 1 {
		return 0x1000
	}
	return 0x0
}
func (ctrl PpuCtrlReg) backgroundPatternAddress() uint16 {
	if (ctrl>>4)&1 == 1 {
		return 0x1000
	}
	return 0x0
}
func (ctrl PpuCtrlReg) spriteSize() uint8 {
	if (ctrl>>5)&1 == 1 {
		return 16 // 8x16
	}
	return 8 // 8x8
}
func (ctrl PpuCtrlReg) vblankNmi() bool {
	return (ctrl>>7)&1 == 1
}

func (mask PpuMaskReg) backgroundOnLeft() bool { return mask&1 == 1 }
func (mask PpuMaskReg) spritesOnLeft() bool    { return (mask>>1)&1 == 1 }
func (mask PpuMaskReg) showBackground() bool   { return (mask>>3)&1 == 1 }
func (mask PpuMaskReg) showSprites() bool      { return (mask>>4)&1 == 1 }

func (status *PpuStatusReg) setSpriteOverflow()   { *status |= 0x20 }
func (status *PpuStatusReg) clearSpriteOverflow() { *status &= 0xdf }
func (status *PpuStatusReg) setSprite0Hit()       { *status |= 0x40 }
func (status *PpuStatusReg) clearSprite0Hit()     { *status &= 0xbf }
func (status *PpuStatusReg) setVblank()           { *status |= 0x80 }
func (status *PpuStatusReg) clearVblank()         { *status &= 0x7f }
