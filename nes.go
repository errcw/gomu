package main

import (
	"fmt"
	"github.com/0xe2-0x9a-0x9b/Go-SDL/sdl"
	"os"
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
	if len(os.Args) != 2 {
		fmt.Println("Usage: gomu [rom]")
		return
	}

	rom, err := LoadRom(os.Args[1])
	if err != nil {
		panic(fmt.Sprintf("Failed to load ROM: %v", err))
	}

	if sdl.Init(sdl.INIT_VIDEO|sdl.INIT_JOYSTICK) != 0 {
		panic(fmt.Sprintf("SDL failed to initialize: %v", sdl.GetError()))
	}
	defer sdl.Quit()

	screen := sdl.SetVideoMode(256, 240, 32, sdl.SWSURFACE)
	if screen == nil {
		panic(fmt.Sprintf("SDL screen failed to initialize: %v", sdl.GetError()))
	}
	sdl.WM_SetCaption("Gomu", "")

	nes := NewNes(rom)

RUN:
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

		// Pump events
		event := sdl.Poll()
		switch event.(type) {
		case sdl.QuitEvent:
			break RUN
		}
	}
}
