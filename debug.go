package main

import (
	"fmt"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	/// Current start address for disassembled instructions.
	///
	Address uint

	/// Create a buffer to hold all logged text.
	///
	LogBuf []string

	/// Current position of the log.
	///
	LogPos int
)

/// Output a new line to the log.
///
func Log(s ...string) {
	t := strings.Join(s, " ")

	if AssembleOnly {
		fmt.Println(t)
	} else {
		scroll := LogPos == len(LogBuf)

		// add the new line
		LogBuf = append(LogBuf, t)

		if scroll {
			LogPos = len(LogBuf)
		}
	}
}

/// Outline a new line to the log, with a newline before it.
///
func Logln(s ...string) {
	t := strings.Join(s, " ")

	if AssembleOnly {
		fmt.Println()
		fmt.Println(t)
	} else {
		scroll := LogPos == len(LogBuf)

		// append the lines
		LogBuf = append(LogBuf, "", t)

		if scroll {
			LogPos = len(LogBuf)
		}
	}
}

/// Show the HELP text in the log.
///
func DebugHelp() {
	Logln("Keys        | Description")
	Log("------------+-------------------------------------")
	Log("BACK        | Reboot (+CTRL to break on reset)")
	Log("[ / ]       | Deacrease/increase speed")
	Log("HOME / END  | Scroll log")
	Log("PGUP / PGDN | Scroll log")
	Log("F2          | Reload ROM / C8 assember")
	Log("F3          | Open ROM / C8 assembler")
	Log("F4          | Save ROM")
	Log("F5          | Pause/break")
	Log("F6          | Step")
	Log("F7          | Step over")
	Log("F8          | Debug memory")
	Log("F9          | Set breakpoint")
}

/// DebugAssembly renders the disassembled instructions around
/// the CHIP-8 program counter.
///
func DebugAssembly(x, y int) {
	if Address <= VM.PC-38 || Address >= VM.PC-2 || (Address&1) != (VM.PC&1) {
		Address = VM.PC-2
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
				X: int32(x-2),
				Y: int32(y + i*5)-1,
				W: 202,
				H: 10,
			})
		}

		DrawText(VM.Disassemble(Address + uint(i)), x, y + i*5)

		// is there a breakpoint on this instruction?
		if _, exists := VM.Breakpoints[int(Address) + i]; exists {
			Renderer.SetDrawColor(255, 0, 0, 255)
			Renderer.DrawRect(&sdl.Rect{
				X: int32(x-2),
				Y: int32(y + i*5)-1,
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
		DrawText(fmt.Sprintf("V%X = #%02X", i, VM.V[i]), x, y + i*10)
	}

	// shift over to next column
	x += 98

	// show the v-registers
	DrawText(fmt.Sprintf("DT = #%02X", VM.GetDelayTimer()), x, y)
	DrawText(fmt.Sprintf("ST = #%02X", VM.GetSoundTimer()), x, y+10)
	DrawText(fmt.Sprintf(" I = #%04X", VM.I), x, y+30)
	DrawText(fmt.Sprintf("PC = #%04X", VM.PC), x, y+50)
	DrawText(fmt.Sprintf("SP = #%02X", VM.SP), x, y+60)

	// show the HP-RPL user flags
	for i := 0;i < 8;i++ {
		DrawText(fmt.Sprintf("R%d = #%02X", i, VM.R[i]), x, y+80 + i*10)
	}
}

/// Show a memory dump at I.
///
func DebugMemory() {
	Logln("Memory dump at I...")

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
		for i := 0;i < 12;i++ {
			if n+i < 0x10000 {
				s[i+1] = fmt.Sprintf("%02X", VM.Memory[n+i])
			} else {
				s[i+1] = ""
			}
		}

		Log(s...)
	}
}

/// Show the current log text (and get new text).
///
func DebugLog(x, y int) {
	line := LogPos - 15
	if line < 0 {
		line = 0
	}

	// display the log
	for i := 0;i < 16 && line < len(LogBuf);i++ {
		if len(LogBuf[line]) >= 54 {
			DrawText(LogBuf[line][:52] + "...", x, y)
		} else {
			DrawText(LogBuf[line], x, y)
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
	if LogPos >= len(LogBuf) {
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
	LogPos = len(LogBuf)
}
