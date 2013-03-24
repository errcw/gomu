package main

import (
	"flag"
	"fmt"
	"github.com/0xe2-0x9a-0x9b/Go-SDL/sdl"
	"github.com/0xe2-0x9a-0x9b/Go-SDL/sdl/audio"
	"math"
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

var keyMap = map[uint32]int{
	sdl.K_UP:     InputUp,
	sdl.K_DOWN:   InputDown,
	sdl.K_LEFT:   InputLeft,
	sdl.K_RIGHT:  InputRight,
	sdl.K_a:      InputA,
	sdl.K_z:      InputB,
	sdl.K_RETURN: InputStart,
	sdl.K_RSHIFT: InputSelect,
}

const (
	ScreenWidth  = 256
	ScreenHeight = 240
)

var scale = 1

func blit(pixels []Pixel, surface *sdl.Surface) {
	surface.Lock()
	surfacePtr := uintptr(surface.Pixels)
	for y := 0; y < ScreenHeight; y++ {
		pixelIndex := y * ScreenWidth
		for sy := 0; sy < scale; sy++ {
			for x := 0; x < ScreenWidth; x++ {
				pixel := pixels[pixelIndex]
				pixelIndex++
				color := sdl.MapRGBA(surface.Format, pixel.R, pixel.G, pixel.B, 255)
				for sx := 0; sx < scale; sx++ {
					*(*uint32)(unsafe.Pointer(surfacePtr)) = color
					surfacePtr += unsafe.Sizeof(color)
				}
			}
			pixelIndex -= ScreenWidth
		}
	}
	surface.Unlock()
	surface.Flip()
}

func runAudio(ch chan []int16) {
	for samples := <-ch; samples != nil; {
		audio.SendAudio_int16(samples)
	}
}

func main() {
	flag.IntVar(&scale, "scale", 1, "scaling factor to apply to the screen size")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Println("Usage: gomu [--scale=<factor>] /path/to/rom")
		return
	}

	rom, err := LoadRom(flag.Arg(0))
	if err != nil {
		panic(fmt.Sprintf("Failed to load ROM: %v", err))
	}

	if sdl.Init(sdl.INIT_VIDEO|sdl.INIT_JOYSTICK|sdl.INIT_AUDIO) != 0 {
		panic(fmt.Sprintf("SDL failed to initialize: %v", sdl.GetError()))
	}
	defer sdl.Quit()

	screen := sdl.SetVideoMode(ScreenWidth*scale, ScreenHeight*scale, 32, sdl.SWSURFACE)
	if screen == nil {
		panic(fmt.Sprintf("SDL screen failed to initialize: %v", sdl.GetError()))
	}
	sdl.WM_SetCaption("Gomu", "")

	audioSpec := &audio.AudioSpec{
		Freq:     44100,
		Format:   audio.AUDIO_S16SYS,
		Channels: 1,
		Samples:  4410,
	}
	if audio.OpenAudio(audioSpec, nil) != 0 {
		panic(fmt.Sprintf("SDL audio failed to initialize: %v", sdl.GetError()))
	}
	defer audio.CloseAudio()
	audio.PauseAudio(false)

	audioChan := make(chan []int16, 2)
	go runAudio(audioChan)

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
		// TODO forward audio to audioChan

		// Pump events
		event := sdl.Poll()
		switch e := event.(type) {
		case sdl.KeyboardEvent:
			if in, ok := keyMap[e.Keysym.Sym]; ok {
				nes.input.SetState(0, in, e.Type == sdl.KEYDOWN)
			}
			if e.Keysym.Sym == sdl.K_ESCAPE {
				break RUN
			}
		case sdl.QuitEvent:
			break RUN
		}
	}
}
