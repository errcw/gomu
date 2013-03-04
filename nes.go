package main

import (
	"fmt"
	"github.com/0xe2-0x9a-0x9b/Go-SDL/sdl"
	"unsafe"
)

type Nes struct {
	cpu   *Cpu
	ppu   *Ppu
	apu   *Apu
	input *Input
	mem   *MemoryMap
}

func NewNes(rom *Rom) *Nes {
	cpu := &Cpu{}
	ppu := &Ppu{}
	apu := &Apu{}
	input := &Input{}
	mem := &MemoryMap{
		cpu:    cpu,
		ppu:    ppu,
		apu:    apu,
		input:  input,
		mapper: NewMapper(rom)}

	cpu.MemoryMap = mem
	cpu.Power()
	cpu.Reset()

	return &Nes{cpu, ppu, apu, input, mem}
}

func blit(pixels []Pixel, surface *sdl.Surface) {
	var pixel uint32
	surface.Lock()
	pixelPtr := uintptr(surface.Pixels)
	for _, p := range pixels {
		*(*uint32)(unsafe.Pointer(pixelPtr)) = sdl.MapRGBA(surface.Format, p.R, p.G, p.B, 255)
		pixelPtr += unsafe.Sizeof(pixel)
	}
	surface.Unlock()
	surface.Flip()
}

func main() {
	if sdl.Init(sdl.INIT_VIDEO|sdl.INIT_JOYSTICK) != 0 {
		panic(fmt.Sprintf("SDL failed to initialize: %v", sdl.GetError()))
	}
	defer sdl.Quit()

	screen := sdl.SetVideoMode(256, 240, 32, sdl.SWSURFACE)
	if screen == nil {
		panic(fmt.Sprintf("SDL screen failed to initialize: %v", sdl.GetError()))
	}

	rom, err := LoadRom("testdata/instr_test-v3/official_only.nes")
	if err != nil {
		panic(fmt.Sprintf("Failed to load ROM: %v", err))
	}

	nes := NewNes(rom)

	steps := 0
	for {
		cycles := nes.cpu.Step()

		ppuResult := nes.ppu.Step(cycles)
		switch ppuResult {
		case PpuVblankNmi:
			nes.cpu.Nmi()
		case PpuNewFrame:
			blit(nes.ppu.framebuffer, screen)
		}

		nes.apu.Step(cycles)

		steps++
		if steps > 10000000 {
			break
		}
	}
}
