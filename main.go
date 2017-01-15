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

const (
	VERSION_MAJOR = 1
	VERSION_MINOR = 0
	VERSION_PATCH = 0
)

var (
	/// AssembleOnly is true if CHIP-8 should only assemble the ROM and
	/// not run it.
	///
	AssembleOnly bool

	/// BreakOnLoad is true if the ROM should load paused.
	///
	BreakOnLoad bool

	/// ETI is true if assembling for an ETI-660 (ROM starts at 0x600
	/// instead of 0x200).
	///
	ETI bool

	/// IncludeInterpreter is true when writing the ROM, include the CDP1802
	/// interpreter.
	///
	IncludeInterpreter bool

	/// Paused is true if emulation is paused (single stepping).
	///
	Paused bool

	/// File is the file that will be/was loaded.
	///
	File string

	/// OutputFile specifies the location on disk to write the final ROM to.
	///
	OutputFile string

	/// VM is the CHIP-8 virtual machine.
	///
	VM *chip8.CHIP_8

	/// The SDL Window and Renderer.
	///
	Window *sdl.Window
	Renderer *sdl.Renderer
)

func init() {
	runtime.LockOSThread()
}

func main() {
	var err error

	// seed the random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// parse the command line
	flag.BoolVar(&AssembleOnly, "a", false, "Assemble the ROM only; do not run it.")
	flag.BoolVar(&BreakOnLoad, "b", false, "Start ROM paused.")
	flag.StringVar(&OutputFile, "o", "", "Write assembled ROM to file.")
	flag.BoolVar(&IncludeInterpreter, "i", false, "When saving the ROM, include the CDP1802 interpreter.")
	flag.BoolVar(&ETI, "eti", false, "Start ROM at 0x600 for ETI-660.")
	flag.Parse()

	// get the file name of the ROM to load
	if File = flag.Arg(0); File == "" && AssembleOnly {
		fmt.Println("Usage: CHIP-8 [-a] [-eti] [-o <bin>] [-b] <ROM|C8>")
		fmt.Println("  -a         assemble/load the ROM only; do not run it")
		fmt.Println("  -eti       assemble/load the ROM in ETI-660 mode")
		fmt.Println("  -o         save the assembled ROM to <file>")
		fmt.Println("  -i         when saving the ROM, include the CDP1802 interpreter")
		fmt.Println("  -b         immediately break on load")

		// exit program
		return
	}

	// note mutually exclusive flags
	if ETI && IncludeInterpreter {
		fmt.Println("Cannot include the CDP1802 interpreter with an ETI-660 ROM.")

		// exit the program
		return
	}

	// initialize SDL or panic if running it
	if !AssembleOnly {
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
	}

	// show copyright information
	Log("CHIP-8, Copyright 2017 by Jeffrey Massung")
	Log("All rights reserved")

	// show the current version number
	Logln("V", fmt.Sprintf("%d.%d.%d", VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH))

	// create a new CHIP-8 virtual machine, which loads/assembles the rom
	if err := Load(); err != nil {
		Log(err.Error())
	} else {
		if OutputFile != "" {
			Save()
		}
	}

	// if running the ROM is desired, do it now
	if !AssembleOnly {
		InitScreen()
		InitAudio()
		InitFont()

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
		Logln("No ROM/C8 specified; loading boot.")
		Log("Press F3 to load a ROM/C8.")

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

			// should the VM start paused?
			Paused = BreakOnLoad

			// clear flag so it doesn't happen on reset
			BreakOnLoad = false
		}
	}

	return err
}

func Save() error {
	file := OutputFile

	if file == "" {
		dlg := dialog.File().Title("Save CHIP-8 ROM")

		dlg.Filter("All Files", "*")
		dlg.Filter("Binary Files", "bin")
		dlg.Filter("ROM Files", "rom")

		// pick a file to save to
		if saveFile, err := dlg.Save(); err != nil {
			Logln(err.Error())

			// don't try and write it
			return err
		} else {
			file = saveFile
		}
	}

	// write the ROM
	err := VM.SaveROM(file, IncludeInterpreter)

	if err == nil {
		Logln("ROM saved to", filepath.Base(file))
	} else {
		Logln(err.Error())
	}

	return err
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
