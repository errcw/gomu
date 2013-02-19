package main

import "fmt"

type Mapper interface {
	Load(addr uint16) uint8
	Store(addr uint16, val uint8)
}

func NewMapper(rom *Rom) Mapper {
	switch rom.Mapper() {
	case 0:
		return Nrom{rom}
	}
	panic("Unimplemented mapper")
}

type Nrom struct {
	rom *Rom
}

func (nrom Nrom) Load(addr uint16) uint8 {
	if addr < 0x8000 {
		panic("Cannot read from low addresses")
	}
	if nrom.rom.header.PrgRomSize > 1 {
		// Map both 16k blocks of PRG ROM
		return nrom.rom.prg[addr&0x7fff]
	}
	// Mirror 16k PRG ROM at 0x8000 and 0xc000
	return nrom.rom.prg[addr&0x3fff]
}

func (nrom Nrom) Store(addr uint16, val uint8) {
	panic(fmt.Sprintf("Cannot write %x to nrom at %x", val, addr))
}
