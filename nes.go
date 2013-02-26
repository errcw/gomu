package main

import "fmt"
import "github.com/0xe2-0x9a-0x9b/Go-SDL/sdl"

func main() {
	fmt.Println("Start")
	if sdl.Init(sdl.INIT_VIDEO) != 0 {
		panic(fmt.Sprintf("SDL failed to init: %v", sdl.GetError()))
	}
	defer sdl.Quit()
	fmt.Println("Init")

	screen := sdl.SetVideoMode(640, 480, 32, sdl.RESIZABLE)
	fmt.Println("Vid")
	if screen == nil {
		panic(fmt.Sprintf("Screen failed to init: %v", sdl.GetError()))
	}

	var video_info = sdl.GetVideoInfo()
	fmt.Println("HW_available = ", video_info.HW_available)
	fmt.Println("WM_available = ", video_info.WM_available)
	fmt.Println("Video_mem = ", video_info.Video_mem, "kb")

	rom, err := LoadRom("testdata/instr_test-v3/official_only.nes")
	if err != nil {
		panic(fmt.Sprintf("Failed to load ROM: %v", err))
	}

	ppu := Ppu{}
	apu := Apu{}
	mem := &MemoryMap{ppu: ppu, apu: apu, mapper: NewMapper(rom)}
	cpu := NewCpu(mem)
	cpu.Reset()

	for {
		cycles := cpu.Step()

		ppuResult := ppu.Step(cycles)
		switch ppuResult {
		case PpuVblankNmi:
			cpu.Nmi()
		case PpuNewFrame:
			// blt
		}

		apu.Step(cycles)
	}
}
