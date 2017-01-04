package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"fmt"
)

var (
	/// Mapping of modern keyboard to CHIP-8 keys.
	///
	KeyMap = map[sdl.Scancode]uint {
		sdl.SCANCODE_X: 0x0,
		sdl.SCANCODE_1: 0x1,
		sdl.SCANCODE_2: 0x2,
		sdl.SCANCODE_3: 0x3,
		sdl.SCANCODE_Q: 0x4,
		sdl.SCANCODE_W: 0x5,
		sdl.SCANCODE_E: 0x6,
		sdl.SCANCODE_A: 0x7,
		sdl.SCANCODE_S: 0x8,
		sdl.SCANCODE_D: 0x9,
		sdl.SCANCODE_Z: 0xA,
		sdl.SCANCODE_C: 0xB,
		sdl.SCANCODE_4: 0xC,
		sdl.SCANCODE_R: 0xD,
		sdl.SCANCODE_F: 0xE,
		sdl.SCANCODE_V: 0xF,
	}
)

/// ProcessEvents from SDL and map keys to the CHIP-8 VM.
///
func ProcessEvents() bool {
	for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
		switch ev := e.(type) {
		case *sdl.QuitEvent:
			return false
		case *sdl.KeyUpEvent:
			if ev.Repeat == 0 {
				switch ev.Keysym.Scancode {
				case sdl.SCANCODE_ESCAPE:
					return false
				case sdl.SCANCODE_UP, sdl.SCANCODE_PAGEUP:
					DebugLogScroll(-1)
				case sdl.SCANCODE_DOWN, sdl.SCANCODE_PAGEDOWN:
					DebugLogScroll(1)
				case sdl.SCANCODE_HOME:
					DebugLogHome()
				case sdl.SCANCODE_END:
					DebugLogEnd()
				case sdl.SCANCODE_F1:
					DebugHelp()
				case sdl.SCANCODE_F9:
					Paused = !Paused
				case sdl.SCANCODE_F10:
					if Paused {
						VM.Step()
					}
				case sdl.SCANCODE_F11:
					DebugMemory()
				case sdl.SCANCODE_F12:
					SaveScreen()
					fmt.Println("Screen saved to SCREENSHOT.BMP")
				case sdl.SCANCODE_BACKSPACE:
					fmt.Println("Rebooting CHIP-8")
					VM.Reset()
				default:
					if key, ok := KeyMap[ev.Keysym.Scancode]; ok {
						VM.ReleaseKey(key)
					}
				}
			}
		case *sdl.KeyDownEvent:
			if ev.Repeat == 0 {
				if key, ok := KeyMap[ev.Keysym.Scancode]; ok {
					VM.PressKey(key)
				}
			}
		}
	}

	return true
}
