package main

import (
	"math/rand"
	"runtime"
	"time"

	"github.com/massung/chip-8/chip8"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	/// The CHIP-8 virtual machine.
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

	// create a new CHIP-8 virtual machine, must happen early!
	VM = chip8.Load("games/BRIX")

	// initialize SDL or panic
	if err = sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		panic(err)
	}

	// create the main window and renderer or panic
	flags := sdl.WINDOW_OPENGL | sdl.WINDOWPOS_CENTERED
	if Window, Renderer, err = sdl.CreateWindowAndRenderer(550, 348, uint32(flags)); err != nil {
		panic(err)
	}

	// set the icon
	if icon, err := sdl.LoadBMP("data/chip_8.bmp"); err == nil {
		mask := sdl.MapRGB(icon.Format, 255, 0, 255)

		// create the mask color key and set the icon
		icon.SetColorKey(1, mask)
		Window.SetIcon(icon)
	}

	// set the title
	Window.SetTitle("CHIP-8")

	// initialize subsystems
	InitScreen()
	InitAudio()
	InitFont()

	// set processor speed and refresh rate
	clock := time.NewTicker(time.Millisecond * 3)
	video := time.NewTicker(time.Second / 60)

	// loop until window closed or user quit
	for ProcessEvents() {
		select {
		case <-video.C:
			Refresh()
		case <-clock.C:
			if !Paused {
				VM.Step()
			}
		}
	}
}

func Refresh() {
	Renderer.SetDrawColor(32, 42, 53, 255)
	Renderer.Clear()

	// frame various portions of the app
	Frame(8, 8, 322, 162)
	Frame(338, 8, 204, 162)
	Frame(8, 176, 146, 164)

	// update the video screen and copy it
	RefreshScreen()
	CopyScreen(10, 10, 5)

	// debug assembly and virtual registers
	DebugAssembly(342, 12)
	DebugRegisters(12, 180)

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
