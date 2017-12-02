/* Copyright (c) 2017 Jeffrey Massung
 *
 * This software is provided 'as-is', without any express or implied
 * warranty.  In no event will the authors be held liable for any damages
 * arising from the use of this software.
 *
 * Permission is granted to anyone to use this software for any purpose,
 * including commercial applications, and to alter it and redistribute it
 * freely, subject to the following restrictions:
 *
 * 1. The origin of this software must not be misrepresented; you must not
 *    claim that you wrote the original software. If you use this software
 *    in a product, an acknowledgment in the product documentation would be
 *    appreciated but is not required.
 *
 * 2. Altered source versions must be plainly marked as such, and must not be
 *    misrepresented as being the original software.
 *
 * 3. This notice may not be removed or altered from any source distribution.
 */

package main

// void Tone(void *data, void *stream, int len);
import "C"
import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"time"
	"unsafe"

	"github.com/massung/CHIP-8/chip8"
	"github.com/sqweek/dialog"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	// VM is the CHIP-8 virtual machine.
	VM *chip8.CHIP_8

	// Window is the global SDL window.
	Window *sdl.Window

	// Renderer is the global SDL renderer.
	Renderer *sdl.Renderer

	// Screen is the global SDL render target for the VM's video memory.
	Screen *sdl.Texture

	// Font is a fixed-width, bitmap font.
	Font *sdl.Texture

	// Debug is the output Logger.
	Debug *Logger

	// ETI is true if ROM starts at 0x600 instead of 0x200.
	ETI bool

	// Paused is true if emulation is paused (single stepping).
	Paused bool

	// File is the currently opened ROM/C8.
	File string

	// Volume is the current tone volume level. When ST is non-zero
	// the volume will be 1.0. But, when ST hits 0 then the volume
	// needs to be ramped down to 0.0.
	Volume float32

	// Address is the current start address for disassembled instructions.
	Address uint

	// KeyMap of modern keyboard keys to CHIP-8 keys.
	KeyMap = map[sdl.Scancode]uint{
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

	// Icon is the compressed bitmap image used for the title bar.
	Icon = []byte{
		0x1F, 0x8B, 0x08, 0x08, 0xCD, 0x5A, 0x79, 0x58,
		0x00, 0x03, 0x63, 0x68, 0x69, 0x70, 0x2D, 0x38,
		0x5F, 0x31, 0x36, 0x2E, 0x62, 0x6D, 0x70, 0x00,
		0x73, 0xF2, 0x35, 0x63, 0x66, 0x00, 0x03, 0x33,
		0x20, 0xD6, 0x00, 0x62, 0x01, 0x28, 0x66, 0x64,
		0x90, 0x80, 0x48, 0x00, 0xE5, 0x85, 0xB8, 0x21,
		0x18, 0x06, 0x4C, 0xB5, 0x14, 0x48, 0x45, 0xBE,
		0xB6, 0x7A, 0x40, 0xF4, 0x1F, 0x0C, 0x20, 0x6C,
		0x64, 0x84, 0x2C, 0x0E, 0x57, 0x0F, 0x17, 0x41,
		0xD3, 0x82, 0x26, 0x8E, 0xA9, 0x1E, 0xD3, 0x4C,
		0xAA, 0x98, 0x4F, 0x92, 0xFB, 0x49, 0x45, 0xAD,
		0x13, 0xFB, 0xD1, 0x90, 0xB6, 0xAC, 0x20, 0x56,
		0x12, 0x97, 0x7A, 0x5C, 0x1A, 0xE1, 0xEA, 0x81,
		0x6C, 0xB8, 0x20, 0x84, 0x81, 0x55, 0x23, 0xB2,
		0xF9, 0xC8, 0x46, 0xE1, 0xB2, 0x0B, 0xCD, 0x7C,
		0x38, 0xC2, 0xA5, 0x85, 0x6C, 0xF5, 0xC8, 0x5A,
		0xF0, 0x78, 0x9F, 0x60, 0xF8, 0x60, 0x55, 0x4F,
		0x12, 0x02, 0x00, 0x6F, 0x19, 0x80, 0x7F, 0x36,
		0x03, 0x00, 0x00,
	}
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		panic(err)
	}

	// create a new debug log
	Debug = NewLog()

	// show copyright information
	Debug.Log("CHIP-8, Copyright 2017 by Jeffrey Massung")
	Debug.Log("All rights reserved")

	// initialize random number generation for VM
	rand.Seed(time.Now().UTC().UnixNano())

	// parse the command line
	flag.BoolVar(&ETI, "eti", false, "Start ROM at 0x600 for ETI-660.")
	flag.Parse()

	// if launching in ETI mode, note that
	if ETI {
		Debug.Logln("Running in ETI-660 mode")
	}

	// create the new VM
	if file := flag.Arg(0); file != "" {
		load(file)
	} else {
		unload()
	}

	// create the main window, renderer, and screen or panic
	createWindow()
	loadFont()
	initAudio()

	// set processor speed and refresh rate
	clock := time.NewTicker(time.Millisecond)
	video := time.NewTicker(time.Second / 60)

	// notify that the main loop has started
	Debug.Logln("Starting program; press 'H' for help")

	// loop until window closed or user quit
	for processEvents() {
		select {
		case <-video.C:
			redraw()
		case <-clock.C:
			res := VM.Process(Paused)

			switch breakpoint := res.(type) {
			case chip8.Breakpoint:
				if !breakpoint.Once {
					Debug.Log()
					Debug.Log(res.Error())
				}

				// break the emulation
				Paused = true
			}
		}
	}
}

// createWindow creates the SDL window and renderer or panics.
func createWindow() {
	var err error

	// window attributes
	flags := sdl.WINDOW_OPENGL

	// create the window and renderer
	Window, Renderer, err = sdl.CreateWindowAndRenderer(614, 380, uint32(flags))
	if err != nil {
		panic(err)
	}

	// set the title
	Window.SetTitle("CHIP-8")

	// load the icon and use it if found
	setIcon()

	// desired screen format and access
	format := sdl.PIXELFORMAT_RGB888
	access := sdl.TEXTUREACCESS_TARGET

	// create a render target for the display
	Screen, err = Renderer.CreateTexture(uint32(format), access, 128, 64)
	if err != nil {
		panic(err)
	}
}

// setIcon unzips the Icon data and sets it on the window.
func setIcon() {
	if gz, err := gzip.NewReader(bytes.NewReader(Icon)); err == nil {
		defer gz.Close()

		// decompress all the bytes
		if icon, err := ioutil.ReadAll(gz); err == nil {
			rw := sdl.RWFromMem(unsafe.Pointer(&icon[0]), len(icon))

			// read the bitmap data and create the icon surface
			if surface, err := sdl.LoadBMPRW(rw, 1); err == nil {
				Window.SetIcon(surface)
			}
		}
	}
}

// initAudio initializes an audio device for the CHIP-8 virtual machine.
func initAudio() {
	spec := &sdl.AudioSpec{
		Freq:     3000,
		Format:   sdl.AUDIO_F32,
		Channels: 1,
		Samples:  32,
		Callback: sdl.AudioCallback(C.Tone),
	}

	// open the device and start playing it
	if err := sdl.OpenAudio(spec, nil); err != nil {
		panic(err)
	}

	// start playing the tone immediately
	sdl.PauseAudio(false)

	// no sound volume
	Volume = 0.0
}

//export Tone
func Tone(_ unsafe.Pointer, stream unsafe.Pointer, length C.int) {
	p := uintptr(stream)
	n := int(length)

	// perform the conversion cast
	buf := *(*[]C.float)(unsafe.Pointer(&reflect.SliceHeader{
		Data: p,
		Len:  n,
		Cap:  n,
	}))

	// get the current time
	now := time.Now().UnixNano()

	// ramp the volume to the desired end
	if now < VM.ST {
		Volume = 1.0
	} else {
		if Volume > 0.0 {
			Volume -= 0.25
		}
	}

	// fill in the data with a constant tone
	for i := 0; i < n; i += 4 {
		buf[i] = C.float(Volume)
	}
}

// loadFont loads the bitmap surface with font on it.
func loadFont() {
	var surface *sdl.Surface
	var err error

	if surface, err = sdl.LoadBMP("font.bmp"); err != nil {
		panic(err)
	}

	// get the magenta color
	mask := sdl.MapRGB(surface.Format, 255, 0, 255)

	// set the mask color key
	surface.SetColorKey(1, mask)

	// create the texture
	if Font, err = Renderer.CreateTextureFromSurface(surface); err != nil {
		panic(err)
	}
}

// processEvents from SDL and map keys to the CHIP-8 VM.
func processEvents() bool {
	for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
		switch ev := e.(type) {
		case *sdl.QuitEvent:
			return false
		case *sdl.DropEvent:
			load(ev.File)
		case *sdl.KeyboardEvent:
			if ev.Type == sdl.KEYUP {
				if key, ok := KeyMap[ev.Keysym.Scancode]; ev.Type == sdl.KEYUP && ok {
					VM.ReleaseKey(key)
				}
			} else {
				if key, ok := KeyMap[ev.Keysym.Scancode]; ok {
					VM.PressKey(key)
				} else {
					switch ev.Keysym.Scancode {
					case sdl.SCANCODE_ESCAPE:
						unload()
					case sdl.SCANCODE_BACKSPACE:
						reboot(ev.Keysym.Mod&sdl.KMOD_CTRL != 0)
					case sdl.SCANCODE_UP, sdl.SCANCODE_PAGEUP:
						Debug.ScrollUp()
					case sdl.SCANCODE_DOWN, sdl.SCANCODE_PAGEDOWN:
						Debug.ScrollDown(16)
					case sdl.SCANCODE_HOME:
						Debug.Home()
					case sdl.SCANCODE_END:
						Debug.End()
					case sdl.SCANCODE_F2:
						if File != "" {
							load(File)
						}
					case sdl.SCANCODE_F3:
						open()
					case sdl.SCANCODE_F4:
						save()
					case sdl.SCANCODE_H:
						help()
					case sdl.SCANCODE_LEFTBRACKET:
						VM.DecSpeed()
					case sdl.SCANCODE_RIGHTBRACKET:
						VM.IncSpeed()
					case sdl.SCANCODE_F5, sdl.SCANCODE_SPACE:
						Paused = !Paused
					case sdl.SCANCODE_F6, sdl.SCANCODE_F10:
						if Paused {
							if VM.StepOverBreakpoint() {
								Paused = false
							} else {
								VM.Step()
							}
						}
					case sdl.SCANCODE_F7, sdl.SCANCODE_F11:
						if Paused {
							if ev.Keysym.Mod&sdl.KMOD_SHIFT != 0 {
								VM.StepOut()
							} else {
								VM.Step()
							}
						}
					case sdl.SCANCODE_F8:
						if Paused {
							dumpMemory()
						}
					case sdl.SCANCODE_F9:
						if Paused {
							VM.ToggleBreakpoint()
						}
					}
				}
			}
		}
	}

	return true
}

// help logs all the keyboard commands.
func help() {
	Debug.Logln("Keys        | Description")
	Debug.Log("------------+-------------------------------------")
	Debug.Log("BACK        | Reboot (CTRL to break on reset)")
	Debug.Log("[ / ]       | Deacrease/increase speed")
	Debug.Log("HOME / END  | Scroll log")
	Debug.Log("PGUP / PGDN | Scroll log")
	Debug.Log("F2          | Reload ROM/C8 assember")
	Debug.Log("F3          | Open ROM/C8 assembler")
	Debug.Log("F4          | Save ROM")
	Debug.Log("F5          | Pause/break")
	Debug.Log("F6 / F10    | Step over")
	Debug.Log("F7 / F11    | Step into (SHIFT to step out)")
	Debug.Log("F8          | Debug memory")
	Debug.Log("F9          | Toggle breakpoint")
}

// save launches a dialog allowing the user to save the current ROM.
func save() error {
	dlg := dialog.File().Title("Save CHIP-8 ROM")

	dlg.Filter("All Files", "*")
	dlg.Filter("Binary Files", "bin")
	dlg.Filter("ROM Files", "rom")

	// pick a file to save to
	if file, err := dlg.Save(); err != nil {
		Debug.Logln(err.Error())

		// don't try and write it
		return err
	} else {
		err := VM.SaveROM(file, false)

		if err == nil {
			Debug.Logln("ROM saved to", filepath.Base(file))
		} else {
			Debug.Logln(err.Error())
		}

		return err
	}
}

// open shows the open file dialog to load ROM/C8 file.
func open() error {
	dlg := dialog.File().Title("Load ROM / C8 Assembler")

	// types of files to load
	dlg.Filter("All Files", "*")
	dlg.Filter("C8 Assembler Files", "c8", "chip8")
	dlg.Filter("ROMs", "rom", "")

	// try and load it
	if file, err := dlg.Load(); err == nil {
		return load(file)
	} else {
		return err
	}
}

// load a ROM/C8 file.
func load(file string) error {
	var err error

	// log what is being loaded
	Debug.Logln("Loading", filepath.Base(file))

	// save the (attempted) loaded file
	File = file

	// attempt to assemble/load the file
	if VM, err = chip8.LoadFile(file, ETI); err != nil {
		Debug.Log(err.Error())

		// load a dummy ROM so something is there
		VM, _ = chip8.LoadROM(chip8.Dummy, false)
	} else {
		Debug.Log(fmt.Sprint(VM.Size), "bytes")
	}

	return err
}

// unload creates a new VM with the boot ROM.
func unload() {
	if VM != nil {
		Debug.Logln("Unloading ROM")
	}

	// create the new VM with the boot ROM
	VM, _ = chip8.LoadROM(chip8.Boot, false)

	// no longer paused
	Paused = false

	// clear the loaded file
	File = ""
}

// reboot the emulator, restarting the loaded virtual machine ROM.
func reboot(breakOnReset bool) {
	Paused = breakOnReset

	// reset registers and memory
	VM.Reset()
}

// dumpMemory shows the next 48 bytes at the I register.
func dumpMemory() {
	Debug.Logln("Memory dump at I...")

	// starting address
	a := int(VM.I)

	// 12 bytes will be written here
	s := make([]string, 20)

	// show 6 lines of 12 bytes each
	for line := 0; line < 6; line++ {
		n := a + line*12

		// memory address
		s[0] = fmt.Sprintf(" %04X -", n)

		// fill in the 12-byte row
		for i := 0; i < 12; i++ {
			if n+i < 0x10000 {
				s[i+1] = fmt.Sprintf("%02X", VM.Memory[n+i])
			} else {
				s[i+1] = ""
			}
		}

		Debug.Log(s...)
	}
}

// updateScreen with the CHIP-8 video memory.
func updateScreen() {
	if err := Renderer.SetRenderTarget(Screen); err != nil {
		panic(err)
	}

	// the background color for the screen
	Renderer.SetDrawColor(143, 145, 133, 255)
	Renderer.Clear()

	// set the pixel color
	Renderer.SetDrawColor(17, 29, 43, 255)

	// redraw only the dimensions of the video
	w, h := VM.GetResolution()

	// the pitch (in bits) is the width, calculate shift
	shift := uint(6 + (w >> 7))

	// draw all the pixels
	for p := 0; p < w*h; p++ {
		if VM.Video[p>>3]&(0x80>>uint(p&7)) != 0 {
			x := int32(p & (w - 1))
			y := int32(p >> shift)

			// render the pixel to the screen
			Renderer.DrawPoint(x, y)
		}
	}

	// restore the render target
	Renderer.SetRenderTarget(nil)
}

// clear the renderer, redraw everything, and present.
func redraw() {
	updateScreen()

	// clear the renderer
	Renderer.SetDrawColor(32, 42, 53, 255)
	Renderer.Clear()

	// frame the screen, instructions, log, and registers
	frame(8, 8, 386, 194)
	frame(8, 208, 386, 164)
	frame(402, 8, 204, 194)
	frame(402, 208, 204, 164)

	// draw the screen, log, instructions, and registers
	drawScreen()
	drawLog()
	drawInstructions()
	drawRegisters()

	// show it
	Renderer.Present()
}

// copyScreen to the render target at a given location.
func drawScreen() {
	vw, vh := VM.GetResolution()

	// source area of the screen target
	src := sdl.Rect{
		W: int32(vw),
		H: int32(vh),
	}

	// stretch the render target to fit
	Renderer.Copy(Screen, &src, &sdl.Rect{X: 10, Y: 10, W: 384, H: 192})
}

// drawText using the bitmap font a string at a given location.
func drawText(s string, x, y int) {
	src := sdl.Rect{W: 5, H: 7}
	dst := sdl.Rect{
		X: int32(x),
		Y: int32(y),
		W: 5,
		H: 7,
	}

	// loop over all the characters in the string
	for _, c := range s {
		if c > 32 && c < 127 {
			src.X = (c - 33) * 6

			// draw the character to the renderer
			Renderer.Copy(Font, &src, &dst)
		}

		// advance
		dst.X += 7
	}
}

// frame draws a highlighted panel to a rectangular area.
func frame(x, y, w, h int32) {
	Renderer.SetDrawColor(0, 0, 0, 255)
	Renderer.DrawLine(x, y, x+w, y)
	Renderer.DrawLine(x, y, x, y+h)

	// highlight
	Renderer.SetDrawColor(95, 112, 120, 255)
	Renderer.DrawLine(x+w, y, x+w, y+h)
	Renderer.DrawLine(x, y+h, x+w, y+h)
}

// drawLog shows the current log window.
func drawLog() {
	x, y := 12, 212

	for i, s := range Debug.Window(16) {
		if len(s) >= 54 {
			drawText(s[:52]+"...", x, y+i*10)
		} else {
			drawText(s, x, y+i*10)
		}
	}
}

// drawInstructions shows the disassembled code and current instruction.
func drawInstructions() {
	x, y := 406, 12

	// determine if the address window needs to move
	if Address <= VM.PC-38 || Address >= VM.PC-2 || (Address&1) != (VM.PC&1) {
		Address = VM.PC - 2
	}

	// show the disassembled instructions
	for i := 0; i < 38; i += 2 {
		if Address+uint(i) == VM.PC {
			if Paused {
				Renderer.SetDrawColor(176, 32, 57, 255)
			} else {
				Renderer.SetDrawColor(57, 102, 176, 255)
			}

			// highlight the current instruction
			Renderer.FillRect(&sdl.Rect{
				X: int32(x - 2),
				Y: int32(y+i*5) - 1,
				W: 202,
				H: 10,
			})
		}

		drawText(VM.Disassemble(Address+uint(i)), x, y+i*5)

		// is there a breakpoint on this instruction?
		if _, exists := VM.Breakpoints[int(Address)+i]; exists {
			Renderer.SetDrawColor(255, 0, 0, 255)
			Renderer.DrawRect(&sdl.Rect{
				X: int32(x - 2),
				Y: int32(y+i*5) - 1,
				W: 202,
				H: 10,
			})
		}
	}
}

// drawRegisters shows the current value of all virtual registers.
func drawRegisters() {
	x, y := 406, 212

	for i := 0; i < 16; i++ {
		drawText(fmt.Sprintf("V%X = #%02X", i, VM.V[i]), x, y+i*10)
	}

	// shift over to next column
	x += 98

	// show the v-registers
	drawText(fmt.Sprintf("DT = #%02X", VM.GetDelayTimer()), x, y)
	drawText(fmt.Sprintf("ST = #%02X", VM.GetSoundTimer()), x, y+10)
	drawText(fmt.Sprintf(" I = #%04X", VM.I), x, y+30)
	drawText(fmt.Sprintf("PC = #%04X", VM.PC), x, y+50)
	drawText(fmt.Sprintf("SP = #%02X", VM.SP), x, y+60)

	// show the HP-RPL user flags
	for i := 0; i < 8; i++ {
		drawText(fmt.Sprintf("R%d = #%02X", i, VM.R[i]), x, y+80+i*10)
	}
}
