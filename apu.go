package main

import "fmt"

type Apu struct {
	status ApuStatus
}

type ApuStatus uint8

func (apu *Apu) Step(cycles int) {
}

func (apu *Apu) Load(addr uint16) uint8 {
	if addr == 0x4015 {
		return apu.readStatus()
	}
	panic(fmt.Sprintf("APU read from %x unsupported", addr))
}

func (apu *Apu) Store(addr uint16, val uint8) {
	switch {
  case addr <= 0x4003:
		// Pulse 1
  case addr <= 0x4007:
		// Pulse 2
  case addr <= 0x400b:
		// Triangle
  case addr <= 0x400f:
		// Noise
  case addr <= 0x4013:
		// DMC
	case addr == 0x4015:
		apu.writeStatus(val)
	case addr == 0x4017:
		// Frame counter
	}
}

func (apu *Apu) readStatus() uint8 {
	return uint8(apu.status)
}

func (apu *Apu) writeStatus(status uint8) {
	apu.status = ApuStatus(status)
}

func (status ApuStatus) pulseEnabled(ch uint) bool { return (status>>ch)&1 == 1 }
func (status ApuStatus) triangleEnabled() bool     { return status&0x04 == 1 }
func (status ApuStatus) noiseEnabled() bool        { return status&0x08 == 1 }
