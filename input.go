package main

import (
	"github.com/veandco/go-sdl2/sdl"
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
		case *sdl.KeyDownEvent:
			if key, ok := KeyMap[ev.Keysym.Scancode]; ok {
				VM.PressKey(key)
			} else {
				switch ev.Keysym.Scancode {
				case sdl.SCANCODE_ESCAPE:
					File = ""

					// go back to the boot program
					Logln("Unloading ROM")
					Load()
				case sdl.SCANCODE_BACKSPACE:
					VM.Reset()

					// holding control during reset will reboot paused
					if ev.Keysym.Mod&sdl.KMOD_CTRL != 0 {
						Paused = true
					}
				case sdl.SCANCODE_UP, sdl.SCANCODE_PAGEUP:
					DebugLogScroll(-1)
				case sdl.SCANCODE_DOWN, sdl.SCANCODE_PAGEDOWN:
					DebugLogScroll(1)
				case sdl.SCANCODE_HOME:
					DebugLogHome()
				case sdl.SCANCODE_END:
					DebugLogEnd()
				case sdl.SCANCODE_F2:
					Load()
				case sdl.SCANCODE_F3:
					LoadDialog()
				case sdl.SCANCODE_F4:
					Save(false)
				case sdl.SCANCODE_H:
					DebugHelp()
				case sdl.SCANCODE_LEFTBRACKET:
					VM.DecSpeed()
				case sdl.SCANCODE_RIGHTBRACKET:
					VM.IncSpeed()
				case sdl.SCANCODE_F5, sdl.SCANCODE_SPACE:
					Paused = !Paused
				case sdl.SCANCODE_F6, sdl.SCANCODE_F10:
					if Paused {
						VM.Step()
					}
				case sdl.SCANCODE_F7, sdl.SCANCODE_F11:
					if Paused {
						VM.SetOverBreakpoint()
						Paused = false
					}
				case sdl.SCANCODE_F8:
					if Paused  {
						DebugMemory()
					}
				case sdl.SCANCODE_F9:
					if Paused {
						VM.ToggleBreakpoint()
					}
				}
			}
		case *sdl.KeyUpEvent:
			if key, ok := KeyMap[ev.Keysym.Scancode]; ok {
				VM.ReleaseKey(key)
			}
		}
	}

	return true
}
