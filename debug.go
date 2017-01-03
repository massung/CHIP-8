package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	/// True if pausing emulation (single stepping).
	///
	Paused bool

	/// Current debug window address.
	///
	Address uint
)

/// DebugAssembly renders the disassembled instructions around
/// the CHIP-8 program counter.
///
func DebugAssembly(x, y int) {
	if Address <= VM.PC - 30 || Address >= VM.PC - 2 || Address ^ VM.PC & 1 == 1 {
		Address = VM.PC - 4
	}

	// show the disassembled instructions
	for i := 0;i < 32;i+=2 {
		if Address + uint(i) == VM.PC - 2 {
			Renderer.SetDrawColor(57, 102, 176, 255)
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

func DebugRegisters(x, y int) {
	for i := 0;i < 16;i++ {
		DrawText(fmt.Sprintf("V%X - #%02X", i, VM.V[i]), x, y + i * 10)
	}

	// shift over for v-registers
	x += 70

	// show the v-registers
	DrawText(fmt.Sprintf("PC - #%04X", VM.PC), x, y)
	DrawText(fmt.Sprintf("SP - #%04X", VM.SP), x, y + 10)
	DrawText(fmt.Sprintf("I  - #%04X", VM.I), x, y + 30)
	DrawText(fmt.Sprintf("DT - #%02X", VM.GetTimer(VM.DT)), x, y + 50)
	DrawText(fmt.Sprintf("ST - #%02X", VM.GetTimer(VM.ST)), x, y + 60)
}
