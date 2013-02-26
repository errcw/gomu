package main

type Apu struct {
}

func (apu *Apu) Step(cycles int) {
}

func (apu *Apu) Load(addr uint16) uint8 {
	return 0
}

func (apu *Apu) Store(addr uint16, val uint8) {
}
