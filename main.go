package main

import (
	"flag"
	"fmt"
	"math/rand"
	"path/filepath"
	"runtime"
	"time"

	"github.com/massung/chip-8/chip8"
	"github.com/sqweek/dialog"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	/// ETI is true if assembling for an ETI-660 (ROM starts at 0x600
	/// instead of 0x200).
	///
	ETI bool

	/// Paused is true if emulation is paused (single stepping).
	///
	Paused bool

	/// File is the file that will be/was loaded.
	///
	File string

	/// VM is the CHIP-8 virtual machine.
	///
	VM *chip8.CHIP_8
)

func init() {
	runtime.LockOSThread()
}

func main() {
	var err error

	// seed the random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// parse the command line
	flag.BoolVar(&ETI, "eti", false, "Start ROM at 0x600 for ETI-660.")
	flag.Parse()

	// get the file name of the ROM to load
	File = flag.Arg(0)

	// initialize SDL or panic if running it
	if err = sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		panic(err)
	}

	// create the main window and renderer or panic
	flags := sdl.WINDOW_OPENGL | sdl.WINDOWPOS_CENTERED
	if Window, Renderer, err = sdl.CreateWindowAndRenderer(614, 380, uint32(flags)); err != nil {
		panic(err)
	}

	// set the icon
	if icon, err := sdl.LoadBMP("chip8_16.bmp"); err == nil {
		mask := sdl.MapRGB(icon.Format, 255, 0, 255)

		// create the mask color key and set the icon
		icon.SetColorKey(1, mask)
		Window.SetIcon(icon)
	}

	// set the title
	Window.SetTitle("CHIP-8")

	// show copyright information
	Log("CHIP-8, Copyright 2017 by Jeffrey Massung")
	Log("All rights reserved")

	// create a new CHIP-8 virtual machine
	Load()

	// if running the ROM is desired, do it now
	InitScreen()
	InitAudio()
	InitFont()

	// if launching in ETI mode, note that
	if ETI {
		Logln("Running in ETI-660 mode")
	}

	// set processor speed and refresh rate
	clock := time.NewTicker(time.Millisecond * 2)
	video := time.NewTicker(time.Second / 60)

	// notify that the main loop has started
	Logln("Starting program; press 'H' for help")

	// loop until window closed or user quit
	for ProcessEvents() {
		select {
		case <-video.C:
			Refresh()
		case <-clock.C:
			res := VM.Process(Paused)

			switch res.(type) {
			case chip8.Breakpoint:
				Log()
				Log(res.Error())

				// break the emulation
				Paused = true
			}
		}
	}
}

func LoadDialog() {
	dlg := dialog.File().Title("Load ROM / C8 Assembler")

	// types of files to load
	dlg.Filter("All Files", "*")
	dlg.Filter("C8 Assembler Files", "c8", "chip8")
	dlg.Filter("ROMs", "rom", "")

	// try and load it
	if file, err := dlg.Load(); err == nil {
		File = file

		// load the file
		Load()
	}
}

func Load() error {
	var err error

	if File == "" {
		Logln("No ROM/C8 specified; loading boot")
		Log("Press F3 to load a ROM/C8")

		// start the boot program
		VM, _ = chip8.LoadROM(chip8.Boot, false)
	} else {
		Logln("Loading", filepath.Base(File))

		if VM, err = chip8.LoadFile(File, ETI); err != nil {
			Log(err.Error())

			// load a dummy ROM so something is there
			VM, _ = chip8.LoadROM(chip8.Dummy, false)
		} else {
			Log(fmt.Sprint(VM.Size), "bytes")
		}
	}

	return err
}

func Save(includeInterpreter bool) error {
	dlg := dialog.File().Title("Save CHIP-8 ROM")

	dlg.Filter("All Files", "*")
	dlg.Filter("Binary Files", "bin")
	dlg.Filter("ROM Files", "rom")

	// pick a file to save to
	if file, err := dlg.Save(); err != nil {
		Logln(err.Error())

		// don't try and write it
		return err
	} else {
		err := VM.SaveROM(file, includeInterpreter)

		if err == nil {
			Logln("ROM saved to", filepath.Base(file))
		} else {
			Logln(err.Error())
		}

		return err
	}
}

func Refresh() {
	Renderer.SetDrawColor(32, 42, 53, 255)
	Renderer.Clear()

	// frame various portions of the app
	Frame(8, 8, 386, 194)
	Frame(8, 208, 386, 164)
	Frame(402, 8, 204, 194)
	Frame(402, 208, 204, 164)

	// update the video screen and copy it
	RefreshScreen()
	CopyScreen(10, 10, 384, 192)

	// debug assembly and virtual registers
	DebugLog(12, 212)
	DebugAssembly(406, 12)
	DebugRegisters(406, 212)

	// show the new frame
	Renderer.Present()
}

func Frame(x, y, w, h int) {
	Renderer.SetDrawColor(0, 0, 0, 255)
	Renderer.DrawLine(x, y, x + w, y)
	Renderer.DrawLine(x, y, x, y + h)

	// highlight
	Renderer.SetDrawColor(95, 112, 120, 255)
	Renderer.DrawLine(x + w, y, x + w, y + h)
	Renderer.DrawLine(x, y + h, x + w, y + h)
}
