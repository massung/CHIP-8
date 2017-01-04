package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	/// True if pausing emulation (single stepping).
	///
	Paused bool

	/// Current debug window address.
	///
	Address uint

	/// Redirected stdout text to a channel.
	///
	LogChan chan string

	/// Create a buffer to hold all logged text.
	///
	Log []string

	/// Current position of the log.
	///
	LogPos int
)

/// Redirect STDOUT text to a log that can be displayed.
///
func InitDebug() {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	// create the log buffer
	LogChan = make(chan string)

	// redirect stdout
	os.Stdout = w

	// spawn a process to capture stdout
	go func() {
		scanner := bufio.NewScanner(r)

		for scanner.Scan() {
			LogChan <- scanner.Text()
		}
	}()
}

/// Show the HELP text in the log.
///
func DebugHelp() {
	fmt.Println("Virtual keys:")
	fmt.Println("  1-2-3-4")
	fmt.Println("  Q-W-E-R")
	fmt.Println("  A-S-D-F")
	fmt.Println("  Z-X-C-V")
	fmt.Println("")
	fmt.Println("Emulation keys:")
	fmt.Println("  ESC      - Quit")
	fmt.Println("  BS       - Reboot")
	fmt.Println("  Pg Up/Dn - Scroll log")
	fmt.Println("  F1       - Help")
	fmt.Println("  F9       - Pause")
	fmt.Println("  F10      - Step")
	fmt.Println("  F12      - Screenshot")
}

/// DebugAssembly renders the disassembled instructions around
/// the CHIP-8 program counter.
///
func DebugAssembly(x, y int) {
	if Address <= VM.PC - 30 || Address >= VM.PC - 2 || Address ^ VM.PC & 1 == 1 {
		Address = VM.PC - 2
	}

	// show the disassembled instructions
	for i := 0;i < 32;i+=2 {
		if Address + uint(i) == VM.PC {
			if Paused {
				Renderer.SetDrawColor(176, 32, 57, 255)
			} else {
				Renderer.SetDrawColor(57, 102, 176, 255)
			}

			// highlight the current instruction
			Renderer.FillRect(&sdl.Rect{
				X: int32(x),
				Y: int32(y + i * 5) - 1,
				W: 200,
				H: 10,
			})
		}

		DrawText(VM.Disassemble(Address + uint(i)), x, y + i * 5)
	}
}

/// Show the current value of all the CHIP-8 registers.
///
func DebugRegisters(x, y int) {
	for i := 0;i < 16;i++ {
		DrawText(fmt.Sprintf("  V%X - #%02X", i, VM.V[i]), x, y + i * 10)
	}

	// shift over for v-registers
	x += 98

	// show the v-registers
	DrawText(fmt.Sprintf("PC - #%04X", VM.PC), x, y)
	DrawText(fmt.Sprintf("SP - #%04X", VM.SP), x, y + 10)
	DrawText(fmt.Sprintf("I  - #%04X", VM.I), x, y + 30)
	DrawText(fmt.Sprintf("DT - #%02X", VM.GetTimer(VM.DT)), x, y + 50)
	DrawText(fmt.Sprintf("ST - #%02X", VM.GetTimer(VM.ST)), x, y + 60)
}

/// Show the current log text (and get new text).
///
func DebugLog(x, y int) {
	select {
	case text := <-LogChan:
		if LogPos == len(Log) - 1 {
			LogPos += 1
		}

		// append the new line to the log
		Log = append(Log, text)
	default:
	}

	// starting line to display for the log
	line := LogPos - 16

	if line < 0 {
		line = 0
	}

	// display 16 lines of the log
	for i := 0;i < 16 && line < len(Log);i++ {
		if len(Log[line]) >= 45 {
			DrawText(Log[line][:42] + "...", x, y)
		} else {
			DrawText(Log[line], x, y)
		}

		// advance to the next line
		y += 10
		line += 1
	}
}

/// Scroll the debug log up/down.
///
func DebugLogScroll(d int) {
	LogPos += d

	// clamp to home
	if LogPos < 0 {
		DebugLogHome()
	}

	// clamp to end
	if LogPos > len(Log) - 1 {
		DebugLogEnd()
	}
}

/// Scroll to the beginning of the log.
///
func DebugLogHome() {
	LogPos = 0
}

/// Scroll to the end of the log.
///
func DebugLogEnd() {
	LogPos = len(Log) - 1
}
