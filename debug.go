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
	fmt.Println()
	fmt.Println("Keys        | Description")
	fmt.Println("------------+-------------------------------------")
	fmt.Println("BACK        | Reboot (+CTRL to break on reset)")
	fmt.Println("[ / ]       | Deacrease/increase speed")
	fmt.Println("HOME / END  | Scroll log")
	fmt.Println("PGUP / PGDN | Scroll log")
	fmt.Println("F3          | Load ROM / C8 assembler")
	fmt.Println("F5          | Pause/break")
	fmt.Println("F6          | Step")
	fmt.Println("F7          | Step over")
	fmt.Println("F8          | Debug memory")
	fmt.Println("F9          | Set breakpoint")
}

/// DebugAssembly renders the disassembled instructions around
/// the CHIP-8 program counter.
///
func DebugAssembly(x, y int) {
	if Address <= VM.PC - 38 || Address >= VM.PC - 2 || Address ^ VM.PC & 1 == 1 {
		Address = VM.PC - 2
	}

	// show the disassembled instructions
	for i := 0;i < 38;i+=2 {
		if Address + uint(i) == VM.PC {
			if Paused {
				Renderer.SetDrawColor(176, 32, 57, 255)
			} else {
				Renderer.SetDrawColor(57, 102, 176, 255)
			}

			// highlight the current instruction
			Renderer.FillRect(&sdl.Rect{
				X: int32(x - 2),
				Y: int32(y + i * 5) - 1,
				W: 202,
				H: 10,
			})
		}

		DrawText(VM.Disassemble(Address + uint(i)), x, y + i * 5)

		// is there a breakpoint on this instruction?
		if _, exists := VM.Breakpoints[int(Address) + i]; exists {
			Renderer.SetDrawColor(255, 0, 0, 255)
			Renderer.DrawRect(&sdl.Rect{
				X: int32(x - 2),
				Y: int32(y + i * 5) - 1,
				W: 202,
				H: 10,
			})
		}
	}
}

/// Show the current value of all the CHIP-8 registers.
///
func DebugRegisters(x, y int) {
	for i := 0;i < 16;i++ {
		DrawText(fmt.Sprintf("V%X = #%02X", i, VM.V[i]), x, y + i * 10)
	}

	// shift over to next column
	x += 98

	// show the v-registers
	DrawText(fmt.Sprintf("PC = #%04X", VM.PC), x, y)
	DrawText(fmt.Sprintf("SP = #%04X", VM.SP), x, y + 10)
	DrawText(fmt.Sprintf(" I = #%04X", VM.I), x, y + 30)
	DrawText(fmt.Sprintf("DT = #%02X", VM.GetDelayTimer()), x, y + 50)
	DrawText(fmt.Sprintf("ST = #%02X", VM.GetSoundTimer()), x, y + 60)

	// show the HP-RPL user flags
	for i := 0;i < 8;i++ {
		DrawText(fmt.Sprintf("R%d = #%02X", i, VM.R[i]), x, y+80+i*10)
	}
}

/// Show a memory dump at I. Useful for sprite debugging.
///
func DebugMemory() {
	a := int(VM.I) & 0xFFF0

	fmt.Println("\nMemory dump near I...")

	// show 8 lines of 12 bytes each
	for line := 0; line < 8; line++ {
		n := a+line*12

		// memory address
		fmt.Printf(" %04X -", n)

		// 12-byte row
		for i := 0;i < 12;i++ {
			if n+i < 0x10000 {
				fmt.Printf(" %02X", VM.Memory[n + i])
			}
		}

		// end of line
		fmt.Println()
	}
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
	line := LogPos - 15
	if line < 0 {
		line = 0
	}

	// display the log
	for i := 0;i < 16 && line < len(Log);i++ {
		if len(Log[line]) >= 52 {
			DrawText(Log[line][:49] + "...", x, y)
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

	// if too low, jump up to end of first screen
	if d > 0 && LogPos < 16 {
		LogPos = 16
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
