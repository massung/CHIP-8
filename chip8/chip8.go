package chip8

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"
)

/// CHIP_8 virtual machine emulator.
///
type CHIP_8 struct {
	/// ROM memory for CHIP-8. This holds the reserved 512 bytes as
	/// well as the program memory. It is a pristine state upon being
	/// loaded that Memory can be reset back to.
	///
	ROM [0x1000]byte

	/// Memory addressable by CHIP-8. The first 512 bytes are reserved
	/// for the font sprites, any RCA 1802 code, and the stack.
	///
	Memory [0x1000]byte

	/// Video memory for CHIP-8 (64x32 bits). Each bit represents a
	/// single pixel. It is stored MSB first. For example, pixel <0,0>
	/// is bit 0x80 of byte 0. 4x the video memory is used for the
	/// CHIP-48, which is 128x64 resolution.
	///
	Video [0x400]byte

	/// PC is the program counter. All programs begin at 0x200.
	///
	PC uint

	/// SP is the stack pointer. The stack is stored at 0x200 and grows
	/// down. It isn't allowed to be more than 16 cells deep.
	///
	SP uint

	/// I is the address register.
	///
	I uint

	/// V are the 16 virtual registers.
	///
	V [16]byte

	/// The delay timer register. It is set to a time (in ns) in the future
	/// and compared against the current time.
	///
	DT int64

	/// The sound timer register. It is set to a time (in ns) in the future
	/// and compared against the current time.
	///
	ST int64

	/// Clock is the time (in ns) when emulation begins.
	///
	Clock int64

	/// Cycles is how many clock cycles have been processed. The RCA 1802
	/// ran at 4-5 MHz, and each instruction took 16-24 clock cycles. Best
	/// estimations are the 1802 could interpret 500 CHIP-8 instructions
	/// per second.
	///
	Cycles int64

	/// W is the wait key (V-register) pointer. When waiting for a key
	/// to be pressed, it will be set to &V[0..F].
	///
	W *byte

	/// Keys hold the current state for the 16-key pad keys.
	///
	Keys [16]bool

	/// True if the CHIP-8 is in high-res (128x64) mode.
	///
	HighRes bool
}

/// Load a ROM from a byte array and return a new CHIP-8 virtual machine.
///
func LoadROM(program []byte) *CHIP_8 {
	if len(program) > 0x1000 - 0x200 {
		panic("Program too large to fit in memory!")
	}

	// create the new CHIP-8 virtual machine
	vm := &CHIP_8{}

	// copy the RCA 1802 512 byte ROM into the CHIP-8
	for i, b := range rca_1802 {
		vm.ROM[i] = b
	}

	// copy the program into the CHIP-8
	for i, b := range program {
		vm.ROM[i+0x200] = b
	}

	// reset the VM memory
	vm.Reset()

	return vm
}

/// Load a ROM file and return a new CHIP-8 virtual machine.
///
func LoadFile(file string) *CHIP_8 {
	program, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	return LoadROM(program)
}

/// Reset the CHIP-8 virtual machine memory.
///
func (vm *CHIP_8) Reset() {
	for i, b := range vm.ROM {
		vm.Memory[i] = b
	}

	// reset video memory
	vm.Video = [0x400]byte{}

	// reset keys
	vm.Keys = [16]bool{}

	// reset program counter and stack pointer
	vm.PC = 0x200
	vm.SP = 0x200

	// reset address register
	vm.I = 0

	// reset virtual registers
	vm.V = [16]byte{}

	// reset timer registers
	vm.DT = 0
	vm.ST = 0

	// reset the clock and cycles executed
	vm.Clock = time.Now().UnixNano()
	vm.Cycles = 0

	// not waiting for a key
	vm.W = nil

	// not in high-res mode
	vm.HighRes = false
}

/// Save the current state of the CHIP-8 virtual machine.
///
func (vm *CHIP_8) Save(file string) {
	// TODO:
}

/// Restore the current state of the CHIP-8 virtual machine.
///
func (vm *CHIP_8) Restore(file string) {
	// TODO:
}

/// PressKey emulates a CHIP-8 key being pressed.
///
func (vm *CHIP_8) PressKey(key uint) {
	if key < 16 {
		vm.Keys[key] = true

		// if waiting for a key, set it now
		if vm.W != nil {
			*vm.W = byte(key)

			// clear wait flag
			vm.W = nil
		}
	}
}

/// ReleaseKey emulates a CHIP-8 key being released.
///
func (vm *CHIP_8) ReleaseKey(key uint) {
	if key < 16 {
		vm.Keys[key] = false
	}
}

/// Converts a CHIP-8 delay timer register to a byte.
///
func (vm *CHIP_8) GetDelayTimer() byte {
	now := time.Now().UnixNano()

	if now < vm.DT {
		return uint8((vm.DT - now) * 60 / 1000000000)
	}

	return 0
}

/// Converts the CHIP-8 sound timer register to a byte.
///
func (vm *CHIP_8) GetSoundTimer() byte {
	now := time.Now().UnixNano()

	if now < vm.ST {
		return uint8((vm.ST - now) * 60 / 1000000000)
	}

	return 0
}

/// GetResolution returns the width and height of the CHIP-8.
///
func (vm *CHIP_8) GetResolution() (uint, uint) {
	if vm.HighRes {
		return 128, 64
	}

	return 64, 32
}

/// Process CHIP-8 emulation. This will execute until the clock is caught up.
///
func (vm *CHIP_8) Process(paused bool) error {
	now := time.Now().UnixNano()

	// calculate how many cycles should have been executed
	count := (now - vm.Clock) * 500 / 1000000000

	// if paused, count cycles without stepping
	if paused {
		vm.Cycles = count
	} else {
		for vm.Cycles < count {
			if err := vm.Step(); err != nil {
				return err
			}

			// if waiting for a key, catch up
			if vm.W != nil {
				vm.Cycles = count
			}
		}
	}

	return nil
}

/// Step the CHIP-8 virtual machine a single instruction.
///
func (vm *CHIP_8) Step() error {
	if vm.W != nil {
		return nil
	}

	// fetch the next instruction
	inst := vm.fetch()

	// 12-bit address operand
	a := inst & 0xFFF

	// byte and nibble operands
	b := byte(inst & 0xFF)
	n := byte(inst & 0xF)

	// x and y register operands
	x := inst >> 8 & 0xF
	y := inst >> 4 & 0xF

	// instruction decoding
	if inst == 0x00E0 {
		vm.cls()
	} else if inst == 0x00EE {
		vm.ret()
	} else if inst == 0x00FE {
		vm.low()
	} else if inst == 0x00FF {
		vm.high()
	} else if inst&0xF000 == 0x0000 {
		vm.sys(a)
	} else if inst&0xF000 == 0x1000 {
		vm.jump(a)
	} else if inst&0xF000 == 0x2000 {
		vm.call(a)
	} else if inst&0xF000 == 0x3000 {
		vm.skipIf(x, b)
	} else if inst&0xF000 == 0x4000 {
		vm.skipIfNot(x, b)
	} else if inst&0xF00F == 0x5000 {
		vm.skipIfXY(x, y)
	} else if inst&0xF000 == 0x6000 {
		vm.loadX(x, b)
	} else if inst&0xF000 == 0x7000 {
		vm.addX(x, b)
	} else if inst&0xF00F == 0x8000 {
		vm.loadXY(x, y)
	} else if inst&0xF00F == 0x8001 {
		vm.or(x, y)
	} else if inst&0xF00F == 0x8002 {
		vm.and(x, y)
	} else if inst&0xF00F == 0x8003 {
		vm.xor(x, y)
	} else if inst&0xF00F == 0x8004 {
		vm.addXY(x, y)
	} else if inst&0xF00F == 0x8005 {
		vm.subXY(x, y)
	} else if inst&0xF00F == 0x8006 {
		vm.shr(x)
	} else if inst&0xF00F == 0x8007 {
		vm.subYX(x, y)
	} else if inst&0xF00F == 0x800E {
		vm.shl(x)
	} else if inst&0xF00F == 0x9000 {
		vm.skipIfNotXY(x, y)
	} else if inst&0xF000 == 0xA000 {
		vm.loadI(a)
	} else if inst&0xF000 == 0xB000 {
		vm.jumpV0(a)
	} else if inst&0xF000 == 0xC000 {
		vm.rnd(x, b)
	} else if inst&0xF000 == 0xD000 {
		vm.drw(x, y, n)
	} else if inst&0xF0FF == 0xE09E {
		vm.skipIfPressed(x)
	} else if inst&0xF0FF == 0xE0A1 {
		vm.skipIfNotPressed(x)
	} else if inst&0xF0FF == 0xF007 {
		vm.loadXDT(x)
	} else if inst&0xF0FF == 0xF00A {
		vm.loadXK(x)
	} else if inst&0xF0FF == 0xF015 {
		vm.loadDTX(x)
	} else if inst&0xF0FF == 0xF018 {
		vm.loadSTX(x)
	} else if inst&0xF0FF == 0xF01E {
		vm.addIX(x)
	} else if inst&0xF0FF == 0xF029 {
		vm.loadF(x)
	} else if inst&0xF0FF == 0xF033 {
		vm.loadB(x)
	} else if inst&0xF0FF == 0xF055 {
		vm.saveRegs(x)
	} else if inst&0xF0FF == 0xF065 {
		vm.loadRegs(x)
	} else {
		return fmt.Errorf("Invalid opcode: %04X", inst)
	}

	// increment the cycle count
	vm.Cycles += 1

	return nil
}

/// Fetch the next 16-bit instruction to execute.
///
func (vm *CHIP_8) fetch() uint {
	i := vm.PC

	// advance the program counter
	vm.PC += 2

	// return the 16-bit instruction
	return uint(vm.Memory[i])<<8 | uint(vm.Memory[i+1])
}

/// Clear the video display memory.
///
func (vm *CHIP_8) cls() {
	for i := range vm.Video {
		vm.Video[i] = 0
	}
}

/// system call an RCA 1802 program at address in ROM.
///
func (vm *CHIP_8) sys(address uint) {
	// TODO:
}

/// call a subroutine at address.
///
func (vm *CHIP_8) call(address uint) {
	if vm.SP < 0x1E0 {
		panic("Stack overflow!")
	}

	// pre-decrement
	vm.SP -= 2

	// push program counter onto stack
	vm.Memory[vm.SP] = byte(vm.PC >> 8 & 0xFF)
	vm.Memory[vm.SP + 1] = byte(vm.PC & 0xFF)

	// jump to address
	vm.PC = address
}

/// return from subroutine.
///
func (vm *CHIP_8) ret() {
	if vm.SP == 0x200 {
		panic("Stack underflow!")
	}

	// restore program counter
	vm.PC = uint(vm.Memory[vm.SP]) << 8 | uint(vm.Memory[vm.SP + 1])

	// post-increment program counter
	vm.SP += 2
}

/// set low res mode.
///
func (vm *CHIP_8) low() {
	vm.HighRes = false
}

/// set high res mode.
///
func (vm *CHIP_8) high() {
	vm.HighRes = true
}

/// jump to address.
///
func (vm *CHIP_8) jump(address uint) {
	vm.PC = address
}

/// jump to address + v0.
///
func (vm *CHIP_8) jumpV0(address uint) {
	vm.PC = address + uint(vm.V[0])
}

/// skip next instruction if vx == n.
///
func (vm *CHIP_8) skipIf(x uint, b byte) {
	if vm.V[x] == b {
		vm.PC += 2
	}
}

/// skip next instruction if vx != n.
///
func (vm *CHIP_8) skipIfNot(x uint, b byte) {
	if vm.V[x] != b {
		vm.PC += 2
	}
}

/// skip next instruction if vx == vy.
///
func (vm *CHIP_8) skipIfXY(x, y uint) {
	if vm.V[x] == vm.V[y] {
		vm.PC += 2
	}
}

/// skip next instruction if vx != vy.
///
func (vm *CHIP_8) skipIfNotXY(x, y uint) {
	if vm.V[x] != vm.V[y] {
		vm.PC += 2
	}
}

/// skip next instruction if key(vx) is pressed.
///
func (vm *CHIP_8) skipIfPressed(x uint) {
	if vm.Keys[vm.V[x]] {
		vm.PC += 2
	}
}

/// skip next instruction if key(vx) is not pressed.
///
func (vm *CHIP_8) skipIfNotPressed(x uint) {
	if !vm.Keys[vm.V[x]] {
		vm.PC += 2
	}
}

/// load n into vx.
///
func (vm *CHIP_8) loadX(x uint, b byte) {
	vm.V[x] = b
}

/// load y into vx.
///
func (vm *CHIP_8) loadXY(x, y uint) {
	vm.V[x] = vm.V[y]
}

/// load delay timer into vx.
///
func (vm *CHIP_8) loadXDT(x uint) {
	vm.V[x] = vm.GetDelayTimer()
}

/// load vx into delay timer.
///
func (vm *CHIP_8) loadDTX(x uint) {
	vm.DT = time.Now().UnixNano() + int64(vm.V[x])*1000000000/60
}

/// load vx into sound timer.
///
func (vm *CHIP_8) loadSTX(x uint) {
	vm.ST = time.Now().UnixNano() + int64(vm.V[x])*1000000000/60
}

/// load vx with next key hit (blocking).
///
func (vm *CHIP_8) loadXK(x uint) {
	vm.W = &vm.V[x]
}

/// load address register.
///
func (vm *CHIP_8) loadI(address uint) {
	vm.I = address
}

/// load address with BCD of vx.
///
func (vm *CHIP_8) loadB(x uint) {
	n := uint16(vm.V[x])
	b := uint16(0)

	// perform 8 shifts
	for i := uint(0); i < 8; i++ {
		if (b>>0)&0xF >= 5 {
			b += 3
		}
		if (b>>4)&0xF >= 5 {
			b += 3 << 4
		}
		if (b>>8)&0xF >= 5 {
			b += 3 << 8
		}

		// apply shift, pull next bit
		b = (b << 1) | (n >> (7 - i) & 1)
	}

	// write to memory
	vm.Memory[vm.I+0] = byte(b>>8) & 0xF
	vm.Memory[vm.I+1] = byte(b>>4) & 0xF
	vm.Memory[vm.I+2] = byte(b>>0) & 0xF
}

/// load font sprite for vx into I.
///
func (vm *CHIP_8) loadF(x uint) {
	vm.I = uint(vm.V[x]) * 5
}

/// or vx with vy into vx.
///
func (vm *CHIP_8) or(x, y uint) {
	vm.V[x] |= vm.V[y]
}

/// and vx with vy into vx.
///
func (vm *CHIP_8) and(x, y uint) {
	vm.V[x] &= vm.V[y]
}

/// xor vx with vy into vx.
///
func (vm *CHIP_8) xor(x, y uint) {
	vm.V[x] ^= vm.V[y]
}

/// shl vx 1 bit, set carry to MSB of vx before shift.
///
func (vm *CHIP_8) shl(x uint) {
	vm.V[0xF] = vm.V[x] >> 7
	vm.V[x] <<= 1
}

/// shr vx 1 bit, set carry to LSB of vx before shift.
///
func (vm *CHIP_8) shr(x uint) {
	vm.V[0xF] = vm.V[x] & 1
	vm.V[x] >>= 1
}

/// add n to vx.
///
func (vm *CHIP_8) addX(x uint, b byte) {
	vm.V[x] += b
}

/// add vy to vx and set carry.
///
func (vm *CHIP_8) addXY(x, y uint) {
	vm.V[x] += vm.V[y]

	if vm.V[x] < vm.V[y] {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}
}

/// add v to i.
///
func (vm *CHIP_8) addIX(x uint) {
	vm.I += uint(vm.V[x])
}

/// subtract vy from vx, set carry if no borrow.
///
func (vm *CHIP_8) subXY(x, y uint) {
	if vm.V[x] >= vm.V[y] {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}

	vm.V[x] -= vm.V[y]
}

/// subtract vx from vy and store in vx, set carry if no borrow.
///
func (vm *CHIP_8) subYX(x, y uint) {
	if vm.V[y] >= vm.V[x] {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}

	vm.V[x] = vm.V[y] - vm.V[x]
}

/// load a random number & n into vx.
///
func (vm *CHIP_8) rnd(x uint, b byte) {
	vm.V[x] = byte(rand.Int31() & int32(b))
}

/// draw a sprite at I to video memory at vx, vy.
///
func (vm *CHIP_8) drw(x, y uint, n byte) {
	c := byte(0)

	// video memory byte and offset
	b := uint(vm.V[x] >> 3)
	i := uint(vm.V[x] & 7)

	// bytes per row
	w, _ := vm.GetResolution()
	p := w >> 3

	// which scan line will it render on
	y = uint(vm.V[y])*p

	// draw each row of the sprite
	for _, s := range vm.Memory[vm.I : vm.I+uint(n)] {
		n := y + b

		// clip pixels that are off screen
		if (n >= 256 && !vm.HighRes) || (n >= 1024 && vm.HighRes) {
			continue
		}

		// origin pixel values
		b0 := vm.Video[n]
		b1 := vm.Video[n+1]

		// xor pixels
		vm.Video[n] ^= s >> i

		// are there pixels overlapping next byte?
		if i > 0 {
			vm.Video[n+1] ^= s << (8 - i)
		}

		// were any pixels turned off?
		c |= b0 & ^vm.Video[n]
		c |= b1 & ^vm.Video[n+1]

		// next scan line
		y += uint(p)
	}

	// set carry flag if any collision occurred
	if c != 0 {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}
}

/// save registers v0..vx to I.
///
func (vm *CHIP_8) saveRegs(x uint) {
	for i := uint(0); i <= x; i++ {
		vm.Memory[vm.I+i] = vm.V[i]
	}
}

/// load registers v0..vx from I.
///
func (vm *CHIP_8) loadRegs(x uint) {
	for i := uint(0); i <= x; i++ {
		vm.V[i] = vm.Memory[vm.I+i]
	}
}
