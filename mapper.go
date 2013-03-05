package main

import "fmt"

type Mapper interface {
	LoadPrg(addr uint16) uint8
	StorePrg(addr uint16, val uint8)
	LoadChr(addr uint16) uint8
	StoreChr(addr uint16, val uint8)
	Mirroring() Mirroring
}

func NewMapper(rom *Rom) Mapper {
	switch rom.Mapper() {
	case 0:
		return NewNrom(rom)
	case 1:
		return NewMmc1(rom)
	}
	panic(fmt.Sprintf("Unimplemented mapper %v", rom.Mapper()))
}

// NROM: No mapping capability
type Nrom struct {
	rom    *Rom
	prgRam []uint8 // 8 KB RAM
}

func NewNrom(rom *Rom) *Nrom {
	return &Nrom{
		rom:    rom,
		prgRam: make([]uint8, 8192)}
}

func (nrom *Nrom) LoadPrg(addr uint16) uint8 {
	if addr < 0x8000 {
		return nrom.prgRam[addr-0x6000]
	}
	if nrom.rom.header.PrgRom16kBanks > 1 {
		// Map both banks
		return nrom.rom.prg[addr&0x7fff]
	}
	// Mirror single bank at 0x8000 and 0xc000
	return nrom.rom.prg[addr&0x3fff]
}

func (nrom *Nrom) StorePrg(addr uint16, val uint8) {
	if addr < 0x6000 || addr >= 0x8000 {
		panic(fmt.Sprintf("Cannot write %x to nrom at %x", val, addr))
	}
	nrom.prgRam[addr-0x6000] = val
}

func (nrom *Nrom) LoadChr(addr uint16) uint8 {
	return nrom.rom.chr[addr]
}

func (nrom *Nrom) StoreChr(addr uint16, val uint8) {
	panic("Nrom cannot write to CHR ROM")
}

func (nrom *Nrom) Mirroring() Mirroring {
	if nrom.rom.header.Flags6&0x1 == 0 {
		return MirrorHorizontal
	}
	return MirrorVertical
}

// MMC1 / SxROM
type Mmc1 struct {
	rom *Rom

	// RAM
	prgRam []uint8
	chrRam []uint8

	// Registers
	ctrl     Mmc1CtrlReg // 0x8000-0x9fff
	chrBank0 uint8       // 0xa000-0xbfff
	chrBank1 uint8       // 0xc000-0xdfff
	prgBank  uint8       // 0xe000-0xffff

	// Register control
	regAccumulator uint8
	regWriteCount  uint8
}

type Mmc1CtrlReg uint8

func (ctrl Mmc1CtrlReg) prgBankMode() uint8 { return uint8(ctrl >> 2 & 3) }
func (ctrl Mmc1CtrlReg) chrBankMode() uint8 { return uint8(ctrl >> 4 & 1) }
func (ctrl Mmc1CtrlReg) mirrorMode() uint8  { return uint8(ctrl & 3) }

func NewMmc1(rom *Rom) *Mmc1 {
	return &Mmc1{
		rom:    rom,
		ctrl:   0xc, // Default 0x8000 PRG switchable
		prgRam: make([]uint8, 8192),
		chrRam: make([]uint8, 8192)}
}

func (mmc1 *Mmc1) LoadPrg(addr uint16) uint8 {
	if addr <= 0x7fff {
		return mmc1.prgRam[addr-0x6000]
	}

	var bank uint8
	switch {
	case addr <= 0xbfff: // First slot 0x8000-0xbfff
		switch mmc1.ctrl.prgBankMode() {
		case 0, 1: // Switch 32k at 0x8000
			bank = mmc1.prgBank & 0xfe
		case 2: // Fix first bank at 0x8000
			bank = 0
		case 3: // Switch bank at 0x8000
			bank = mmc1.prgBank
		}
	case addr <= 0xffff: // Second slot 0xc000-0xffff
		switch mmc1.ctrl.prgBankMode() {
		case 0, 1: // Switch 32k at 0x8000
			bank = (mmc1.prgBank & 0xfe) | 1
		case 2: // Switch bank at 0xc000
			bank = mmc1.prgBank
		case 3: // Fix last bank at 0xc000
			bank = mmc1.rom.header.PrgRom16kBanks - 1
		}
	}

	return mmc1.rom.prg[(uint16(bank)*0x4000)|(addr&0x3fff)]
}

func (mmc1 *Mmc1) StorePrg(addr uint16, val uint8) {
	if addr >= 0x6000 && addr < 0x8000 {
		mmc1.prgRam[addr-0x6000] = val
		return
	}

	if val&0x80 == 0x80 {
		mmc1.regAccumulator = 0
		mmc1.regWriteCount = 0
		mmc1.ctrl |= 0xc
		return
	}

	mmc1.regAccumulator |= (val & 1) << mmc1.regWriteCount
	mmc1.regWriteCount++
	if mmc1.regWriteCount == 5 {
		switch {
		case addr <= 0x9fff:
			mmc1.ctrl = Mmc1CtrlReg(mmc1.regAccumulator)
		case addr <= 0xbfff:
			mmc1.chrBank0 = mmc1.regAccumulator
		case addr <= 0xdfff:
			mmc1.chrBank1 = mmc1.regAccumulator
		case addr <= 0xffff:
			mmc1.prgBank = mmc1.regAccumulator
		}
		mmc1.regAccumulator = 0
		mmc1.regWriteCount = 0
	}
}

func (mmc1 *Mmc1) LoadChr(addr uint16) uint8 {
	if mmc1.rom.header.ChrRom8kBanks == 0 {
		return mmc1.chrRam[addr]
	}

	var bank uint8
	switch {
	case addr < 0x1000:
		bank = mmc1.chrBank0
	case addr < 0x2000:
		switch bankMode := (mmc1.ctrl >> 4) & 1; bankMode {
		case 0: // 8k
			bank = mmc1.chrBank0 + 1
		case 1: // 4k
			bank = mmc1.chrBank1
		}
	}

	return mmc1.rom.chr[(uint16(bank)*0x1000)|(addr&0xfff)]
}

func (mmc1 *Mmc1) StoreChr(addr uint16, val uint8) {
	mmc1.chrRam[addr] = val
}

func (mmc1 *Mmc1) Mirroring() Mirroring {
	switch mmc1.ctrl.mirrorMode() {
	case 0:
		return MirrorSingleUpper
	case 1:
		return MirrorSingleLower
	case 2:
		return MirrorVertical
	case 3:
		return MirrorHorizontal
	}
	panic("Impossible mirror mode")
}
