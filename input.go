package main

import "fmt"

type Input struct {
	controllers [2]ControllerState
	lastWrite   uint8
}

type ControllerState struct {
	state       [InputMax]bool // Down (true) or up (false) for each button in strobe order
	strobeIndex int            // Current strobe position
}

// Defines the strobe state in which order the NES reads the controller buttons
const (
	InputA = iota
	InputB
	InputSelect
	InputStart
	InputUp
	InputDown
	InputLeft
	InputRight
	InputMax
)

func (input *Input) SetState(controller int, button int, down bool) {
	input.controllers[controller].state[button] = down
}

func (input *Input) Load(addr uint16) uint8 {
	var controller *ControllerState
	switch addr {
	case 0x4016:
		controller = &input.controllers[0]
	case 0x4017:
		controller = &input.controllers[1]
	default:
		panic(fmt.Sprintf("Unexpected address for input load: %x", addr))
	}

	val := uint8(0)
	if controller.strobeIndex < InputMax {
		if controller.state[controller.strobeIndex] {
			val = 0x41
		} else {
			val = 0x40
		}
	}
	controller.strobeIndex++

	return val
}

func (input *Input) Store(addr uint16, val uint8) {
	if addr != 0x4016 {
		panic(fmt.Sprintf("Unexpected address for input store: %x", addr))
	}
	if (val&1) == 0 && (input.lastWrite&1) == 1 {
		input.controllers[0].strobeIndex = 0
		input.controllers[1].strobeIndex = 0
	}
	input.lastWrite = val
}
