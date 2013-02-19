package main

import (
	"encoding/binary"
	"errors"
	"os"
)

type INesHeader struct {
	Magic      [4]byte
	PrgRomSize byte
	ChrRomSize byte
	Flags6     byte
	Flags7     byte
	PrgRamSize byte
	Flags9     byte
	Flags10    byte
	Zero       [5]byte
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

	prgBytes := int(header.PrgRomSize) * 16384
	chrBytes := int(header.ChrRomSize) * 8192

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
