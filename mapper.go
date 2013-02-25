package main

import "fmt"

type Mapper interface {
	Load(addr uint16) uint8
	Store(addr uint16, val uint8)
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

func (nrom *Nrom) Load(addr uint16) uint8 {
	if addr < 0x8000 {
		return nrom.prgRam[addr-0x6000]
	}
	if nrom.rom.header.PrgRom16kBanks > 1 {
		// Map both 16k blocks of PRG ROM
		return nrom.rom.prg[addr&0x7fff]
	}
	// Mirror 16k PRG ROM at 0x8000 and 0xc000
	return nrom.rom.prg[addr&0x3fff]
}

func (nrom *Nrom) Store(addr uint16, val uint8) {
	if addr < 0x6000 || addr >= 0x8000 {
		panic(fmt.Sprintf("Cannot write %x to nrom at %x", val, addr))
	}
	nrom.prgRam[addr-0x6000] = val
}

// MMC1 / SxROM
type Mmc1 struct {
	rom *Rom

	// RAM
	prgRam []uint8
	chrRam []uint8

	// Registers
	ctrl     uint8 // 0x8000-0x9fff
	chrBank0 uint8 // 0xa000-0xbfff
	chrBank1 uint8 // 0xc000-0xdfff
	prgBank  uint8 // 0xe000-0xffff

	// Register control
	regAccumulator uint8
	regWriteCount  uint8
}

const (
	ChrMode8k = iota
	ChrMode4k
)

const (
	MirrorOneScreenLower = iota
	MirrorOneScreenUpper
	MirrorVertical
	MirrorHorizontal
)

func NewMmc1(rom *Rom) *Mmc1 {
	return &Mmc1{
		rom:    rom,
		ctrl:   0xc, // Default 0x8000 PRG switchable
		prgRam: make([]uint8, 8192),
		chrRam: make([]uint8, 8192)}
}

func (mmc1 *Mmc1) Load(addr uint16) uint8 {
	if addr <= 0x7fff {
		fmt.Printf("Loading RAM from %x\n", addr)
		return mmc1.prgRam[addr-0x6000]
	}

	bankMode := (mmc1.ctrl >> 2) & 3
	var bank uint8
	switch {
	case addr <= 0xbfff: // First slot 0x8000-0xbfff
		switch bankMode {
		case 0, 1: // Switch 32k at 0x8000
			bank = mmc1.prgBank & 0xfe
		case 2: // Fix first bank at 0x8000
			bank = 0
		case 3: // Switch bank at 0x8000
			bank = mmc1.prgBank
		}
	case addr <= 0xffff: // Second slot 0xc000-0xffff
		switch bankMode {
		case 0, 1: // Switch 32k at 0x8000
			bank = (mmc1.prgBank & 0xfe) | 1
		case 2: // Switch bank at 0xc000
			bank = mmc1.prgBank
		case 3: // Fix last bank at 0xc000
			bank = mmc1.rom.header.PrgRom16kBanks - 1
		}
	}

	fmt.Printf("Loading %x from bank %x and addr %x\n", bank, addr, mmc1.rom.prg[(uint16(bank)*0x4000)|(addr&0x3fff)])
	return mmc1.rom.prg[(uint16(bank)*0x4000)|(addr&0x3fff)]
}

func (mmc1 *Mmc1) Store(addr uint16, val uint8) {
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
			mmc1.ctrl = mmc1.regAccumulator
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
