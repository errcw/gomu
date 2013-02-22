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
	fmt.Printf("Writing %x (%q) to %x\n", val, val, addr)
	nrom.prgRam[addr-0x6000] = val
}
