package main

type Ppu struct {
}

const (
	PpuTick = iota
	PpuVblankNmi
	PpuNewFrame
)

func (ppu *Ppu) Step(cycles int) int {
	return PpuTick
}

func (ppu *Ppu) Load(addr uint16) uint8 {
	return 0
}

func (ppu *Ppu) Store(addr uint16, val uint8) {
}
