package main

import "fmt"

type Mapper interface {
	Load(addr uint16) uint8
	Store(addr uint16, val uint8)
}

func NewMapper(rom *Rom) Mapper {
	switch rom.Mapper() {
	case 0:
		return &Nrom{rom, make([]uint8, 8192)}
	case 1:
		return &Mmc1{rom: rom, prgRam: make([]uint8, 8192), chrRam: make([]uint8, 8192)}
	}
	panic(fmt.Sprintf("Unimplemented mapper %v", rom.Mapper()))
}

// NROM: No mapping capability
type Nrom struct {
	rom    *Rom
	prgRam []uint8 // 8 KB RAM
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
	PrgSize32k = iota
	PrgSize16k
)

// TODO other ctrl consts

func (mmc1 *Mmc1) Load(addr uint16) uint8 {
	return 0
}

func (mmc1 *Mmc1) Store(addr uint16, val uint8) {
	if addr < 0x8000 {
		// TODO PRG RAM
		return
	}

	if val&0x80 == 0x80 {
		mmc1.regAccumulator = 0
		mmc1.regWriteCount = 0
		mmc1.ctrl |= 3 << 2
	}

	// TODO accum
}
