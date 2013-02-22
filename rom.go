package main

import (
	"encoding/binary"
	"errors"
	"os"
)

type INesHeader struct {
	Magic          [4]byte
	PrgRom16kBanks byte
	ChrRom8kBanks  byte
	Flags6         byte
	Flags7         byte
	PrgRam8kBanks  byte
	Flags9         byte
	Flags10        byte
	Zero           [5]byte
}

type Rom struct {
	header INesHeader
	prg    []byte
	chr    []byte
}

func LoadRom(filename string) (*Rom, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	header := &INesHeader{}
	err = binary.Read(file, binary.LittleEndian, header)
	if err != nil {
		return nil, err
	}

	if string(header.Magic[0:3]) != "NES" {
		return nil, errors.New("ines header corrupted")
	}

	prgBytes := int(header.PrgRom16kBanks) * 16 * 1024
	chrBytes := int(header.ChrRom8kBanks) * 8 * 1024

	rom := &Rom{*header, make([]byte, prgBytes), make([]byte, chrBytes)}

	n, err := file.Read(rom.prg)
	if n != prgBytes || err != nil {
		return nil, errors.New("failed to read prg")
	}

	n, err = file.Read(rom.chr)
	if n != chrBytes || err != nil {
		return nil, errors.New("failed to read chr")
	}

	return rom, nil
}

func (rom Rom) Mapper() uint8 {
	return rom.header.Flags7&0xf0 | rom.header.Flags6>>4
}
