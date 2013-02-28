package main

import (
	"fmt"
	"github.com/0xe2-0x9a-0x9b/Go-SDL/sdl"
	"unsafe"
)

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

	ppu := Ppu{}
	apu := Apu{}
	mem := &MemoryMap{ppu: ppu, apu: apu, mapper: NewMapper(rom)}
	cpu := NewCpu(mem)
	cpu.Reset()

	steps := 0
	for {
		cycles := cpu.Step()

		ppuResult := ppu.Step(cycles)
		switch ppuResult {
		case PpuVblankNmi:
			cpu.Nmi()
		case PpuNewFrame:
			blit(ppu.framebuffer, screen)
		}

		apu.Step(cycles)

		steps++
		if steps > 10000000 {
			break
		}
	}
}
