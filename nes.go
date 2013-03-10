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
	mapper := NewMapper(rom)

	cpu := &Cpu{}
	ppu := &Ppu{vram: &VramMemoryMap{mapper: mapper}}
	apu := &Apu{}
	input := &Input{}
	mem := &MemoryMap{
		cpu:    cpu,
		ppu:    ppu,
		apu:    apu,
		input:  input,
		mapper: mapper}

	ppu.Setup()

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

	//steps := 0
	for {
		cycles := nes.cpu.Step()

		for i := 0; i < cycles*3; i++ {
			ppuResult := nes.ppu.Step()
			switch ppuResult {
			case PpuVblankNmi:
				nes.cpu.Nmi()
			case PpuNewFrame:
				blit(nes.ppu.Framebuffer, screen)
			}
		}

		nes.apu.Step(cycles)

		ram := nes.cpu.MemoryMap.mapper.(*Mmc1).prgRam
		if ram[1] == 0xde && ram[2] == 0xb0 && ram[3] == 0x61 && ram[0] != 0x80 {
			fmt.Println("Breaking--test done")
			break
		}

		/*
					steps++
					if steps > 10000000 {
			      fmt.Println("Breaking--too many steps")
						break
					}
		*/
	}
}
