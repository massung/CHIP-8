package chip8

import "fmt"

/// Disassemble a CHIP-8 instruction.
///
func (vm *CHIP_8) Disassemble(i uint) string {
	if int(i) >= len(vm.Memory) - 1 {
		return ""
	}

	// fetch the instruction at this location
	inst := uint(vm.Memory[i])<<8 | uint(vm.Memory[i+1])

	// end of program memory?
	if inst == 0 {
		return fmt.Sprintf("%04X -", i)
	}

	// 12-bit literal address
	a := inst & 0xFFF

	// byte and nibble literals
	b := byte(inst & 0xFF)
	n := byte(inst & 0xF)

	// vx and vy registers
	x := inst >> 8 & 0xF
	y := inst >> 4 & 0xF

	// instruction decoding
	if inst == 0x00E0 {
		return fmt.Sprintf("%04X - CLS", i)
	} else if inst == 0x00EE {
		return fmt.Sprintf("%04X - RET", i)
	} else if inst&0xF000 == 0x0000 {
		return fmt.Sprintf("%04X - SYS    #%04X", i, a)
	} else if inst&0xF000 == 0x1000 {
		return fmt.Sprintf("%04X - JP     #%04X", i, a)
	} else if inst&0xF000 == 0x2000 {
		return fmt.Sprintf("%04X - CALL   #%04X", i, a)
	} else if inst&0xF000 == 0x3000 {
		return fmt.Sprintf("%04X - SE     V%X, #%02X", i, x, b)
	} else if inst&0xF000 == 0x4000 {
		return fmt.Sprintf("%04X - SNE    V%X, #%02X", i, x, b)
	} else if inst&0xF00F == 0x5000 {
		return fmt.Sprintf("%04X - SE     V%X, V%X", i, x, y)
	} else if inst&0xF000 == 0x6000 {
		return fmt.Sprintf("%04X - LD     V%X, #%02X", i, x, b)
	} else if inst&0xF000 == 0x7000 {
		return fmt.Sprintf("%04X - ADD    V%X, #%02X", i, x, b)
	} else if inst&0xF00F == 0x8000 {
		return fmt.Sprintf("%04X - LD     V%X, V%X", i, x, y)
	} else if inst&0xF00F == 0x8001 {
		return fmt.Sprintf("%04X - OR     V%X, V%X", i, x, y)
	} else if inst&0xF00F == 0x8002 {
		return fmt.Sprintf("%04X - AND    V%X, V%X", i, x, y)
	} else if inst&0xF00F == 0x8003 {
		return fmt.Sprintf("%04X - XOR    V%X, V%X", i, x, y)
	} else if inst&0xF00F == 0x8004 {
		return fmt.Sprintf("%04X - ADD    V%X, V%X", i, x, y)
	} else if inst&0xF00F == 0x8005 {
		return fmt.Sprintf("%04X - SUB    V%X, V%X", i, x, y)
	} else if inst&0xF00F == 0x8006 {
		return fmt.Sprintf("%04X - SHR    V%X", i, x)
	} else if inst&0xF00F == 0x8007 {
		return fmt.Sprintf("%04X - SUBN   V%X, V%X", i, x, y)
	} else if inst&0xF00F == 0x800E {
		return fmt.Sprintf("%04X - SHL    V%X", i, x)
	} else if inst&0xF00F == 0x9000 {
		return fmt.Sprintf("%04X - SNE    V%X, V%X", i, x, y)
	} else if inst&0xF000 == 0xA000 {
		return fmt.Sprintf("%04X - LD     I, V%X", i, x)
	} else if inst&0xF000 == 0xB000 {
		return fmt.Sprintf("%04X - JP     V0, #%04X", i, a)
	} else if inst&0xF000 == 0xC000 {
		return fmt.Sprintf("%04X - RND    V%X, #%02X", i, x, b)
	} else if inst&0xF000 == 0xD000 {
		return fmt.Sprintf("%04X - DRW    V%X, V%X, %d", i, x, y, n)
	} else if inst&0xF0FF == 0xE09E {
		return fmt.Sprintf("%04X - SKP    V%X", i, x)
	} else if inst&0xF0FF == 0xE0A1 {
		return fmt.Sprintf("%04X - SKNP   V%X", i, x)
	} else if inst&0xF0FF == 0xF007 {
		return fmt.Sprintf("%04X - LD     V%X, DT", i, x)
	} else if inst&0xF0FF == 0xF00A {
		return fmt.Sprintf("%04X - LD     V%X, K", i, x)
	} else if inst&0xF0FF == 0xF015 {
		return fmt.Sprintf("%04X - LD     DT, V%X", i, x)
	} else if inst&0xF0FF == 0xF018 {
		return fmt.Sprintf("%04X - LD     ST, V%X", i, x)
	} else if inst&0xF0FF == 0xF01E {
		return fmt.Sprintf("%04X - ADD    I, V%X", i, x)
	} else if inst&0xF0FF == 0xF029 {
		return fmt.Sprintf("%04X - LD     F, V%X", i, x)
	} else if inst&0xF0FF == 0xF033 {
		return fmt.Sprintf("%04X - LD     B, V%X", i, x)
	} else if inst&0xF0FF == 0xF055 {
		return fmt.Sprintf("%04X - LD     [I], V%X", i, x)
	} else if inst&0xF0FF == 0xF065 {
		return fmt.Sprintf("%04X - LD     V%X, [I]", i, x)
	}

	// unknown instruction
	return fmt.Sprintf("%04X - ??", i)
}