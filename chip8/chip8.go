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
	/// CHIP-48, which is 128x64 resolution. There is an extra byte
	/// to prevent overflows.
	///
	Video [0x401]byte

	/// The stack was in a reserved section of memory on the 1802.
	/// Originally it was only 12-cells deep, but later implementations
	/// went as high as 16-cells.
	///
	Stack [16]uint

	/// PC is the program counter. All programs begin at 0x200.
	///
	PC uint

	/// SP is the stack pointer.
	///
	SP uint

	/// I is the Address register.
	///
	I uint

	/// V are the 16 virtual registers.
	///
	V [16]byte

	/// R are the 8, HP-RPL user flags.
	///
	R [8]byte

	/// DT is the delay timer register. It is set to a time (in ns) in the
	/// future and compared against the current time.
	///
	DT int64

	/// ST is the sound timer register. It is set to a time (in ns) in the
	/// future and compared against the current time.
	///
	ST int64

	/// Clock is the time (in ns) when emulation begins.
	///
	Clock int64

	/// Cycles is how many clock cycles have been processed. It is assumed
	/// once clock cycle per instruction.
	///
	Cycles int64

	/// Speed is how many cycles (instructions) should execute per second.
	/// By default this is 1000. The RCA CDP1802 ran at 3.2 MHz, with each
	/// instruction taking 16-24 clock cycles.
	///
	Speed int64

	/// W is the wait key (V-register) pointer. When waiting for a key
	/// to be pressed, it will be set to &V[0..F].
	///
	W *byte

	/// Keys hold the current state for the 16-key pad keys.
	///
	Keys [16]bool

	/// Number of bytes per scan line. This is 8 in low mode and 16 when high.
	///
	Pitch uint

	/// A mapping of instruction Address breakpoints.
	///
	Breakpoints map[int]string
}

/// Breakpoint is an implementation of error.
///
type Breakpoint struct {
	Address int
	Reason  string
}

/// Error implements the error interface for a Breakpoint.
///
func (b Breakpoint) Error() string {
	return fmt.Sprintf("hit breakpoint @ %04X: %s", b.Address, b.Reason)
}

/// SysCall is an implementation of error.
///
type SysCall struct {
	address uint
}

/// Error implements the error interface for a SysCall.
///
func (call SysCall) Error() string {
	return fmt.Sprintf("unimplmented syscall to #%04X", call.address)
}

/// Load a ROM from a byte array and return a new CHIP-8 virtual machine.
///
func LoadROM(program []byte) (*CHIP_8, int) {
	if len(program) > 0x1000 - 0x200 {
		panic("Program too large to fit in memory!")
	}

	// initialize any data that doesn't Reset()
	vm := &CHIP_8{
		Breakpoints: make(map[int]string),
		Speed: 1000,
	}

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

	return vm, len(program)
}

/// Load a ROM file and return a new CHIP-8 virtual machine.
///
func LoadFile(file string) (*CHIP_8, int) {
	program, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	return LoadROM(program)
}

/// Load a compiled assembly and return a new CHIP-8 virtual machine.
///
func LoadAssembly(asm *Assembly) (*CHIP_8, int) {
	vm, size := LoadROM(asm.ROM)

	// set all the breakpoints from the assembly
	for _, b := range asm.Breakpoints {
		vm.SetBreakpoint(b.Address, b.Reason)
	}

	return vm, size
}

/// Reset the CHIP-8 virtual machine memory.
///
func (vm *CHIP_8) Reset() {
	for i, b := range vm.ROM {
		vm.Memory[i] = b
	}

	// reset video memory
	vm.Video = [0x401]byte{}

	// reset keys
	vm.Keys = [16]bool{}

	// reset program counter and stack pointer
	vm.PC = 0x200
	vm.SP = 0

	// reset Address register
	vm.I = 0

	// reset virtual registers
	vm.V = [16]byte{}

	// reset HP-RPL user flags
	vm.R = [8]byte{}

	// reset timer registers
	vm.DT = 0
	vm.ST = 0

	// reset the clock and cycles executed
	vm.Clock = time.Now().UnixNano()
	vm.Cycles = 0

	// not waiting for a key
	vm.W = nil

	// not in high-res mode
	vm.Pitch = 8
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

/// HighRes returns true if the CHIP-8 is in high resolution mode.
///
func (vm *CHIP_8) HighRes() bool {
	return vm.Pitch > 8
}

/// IncSpeed increases CHIP-8 virtual machine performance.
///
func (vm *CHIP_8) IncSpeed() {
	if vm.Speed < 2000 {
		vm.Speed += 200

		// reset the clock
		vm.Clock = time.Now().UnixNano()
		vm.Cycles = 0
	}
}

/// DecSpeed lowers CHIP-8 virtual machine performance.
///
func (vm *CHIP_8) DecSpeed() {
	if vm.Speed > 200 {
		vm.Speed -= 200

		// reset the clock
		vm.Clock = time.Now().UnixNano()
		vm.Cycles = 0
	}
}

/// SetBreakpoint at a ROM Address to the CHIP-8 virtual machine.
///
func (vm *CHIP_8) SetBreakpoint(address int, reason string) {
	if address >= 0x200 && address < len(vm.ROM) {
		vm.Breakpoints[address] = reason
	}
}

/// RemoveBreakpoint clears a breakpoint at a given ROM Address.
///
func (vm *CHIP_8) RemoveBreakpoint(address int) {
	delete(vm.Breakpoints, address)
}

/// ToggleBreakpoint at the current PC. Any reason is lost.
///
func (vm *CHIP_8) ToggleBreakpoint() {
	a := int(vm.PC)

	if _, ok := vm.Breakpoints[a]; !ok {
		vm.SetBreakpoint(a, "User break")
	} else {
		vm.RemoveBreakpoint(a)
	}
}

/// ClearBreakpoints removes all breakpoints.
///
func (vm *CHIP_8) ClearBreakpoints() {
	vm.Breakpoints = make(map[int]string)
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
	return vm.Pitch<<3, vm.Pitch<<2
}

/// Process CHIP-8 emulation. This will execute until the clock is caught up.
///
func (vm *CHIP_8) Process(paused bool) error {
	now := time.Now().UnixNano()

	// calculate how many cycles should have been executed
	count := (now - vm.Clock) * vm.Speed / 1000000000

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

	// 12-bit Address operand
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
	} else if inst == 0x00FB {
		vm.scrollRight()
	} else if inst == 0x00FC {
		vm.scrollLeft()
	} else if inst == 0x00FD {
		vm.exit()
	} else if inst == 0x00FE {
		vm.low()
	} else if inst == 0x00FF {
		vm.high()
	} else if inst&0xFFF0 == 0x00B0 {
		vm.scrollUp(n)
	} else if inst&0xFFF0 == 0x00C0 {
		vm.scrollDown(n)
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
		vm.loadRandom(x, b)
	} else if inst&0xF00F == 0xD000 {
		vm.drawSpriteEx(x, y)
	} else if inst&0xF000 == 0xD000 {
		vm.drawSprite(x, y, n)
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
	} else if inst&0xF0FF == 0xF030 {
		vm.loadHF(x)
	} else if inst&0xF0FF == 0xF033 {
		vm.loadB(x)
	} else if inst&0xF0FF == 0xF055 {
		vm.saveRegs(x)
	} else if inst&0xF0FF == 0xF065 {
		vm.loadRegs(x)
	} else if inst&0xF0FF == 0xF075 {
		vm.storeR(x)
	} else if inst&0xF0FF == 0xF085 {
		vm.readR(x)
	} else {
		return fmt.Errorf("Invalid opcode: %04X", inst)
	}

	// increment the cycle count
	vm.Cycles += 1

	// if at a breakpoint, return it
	if s, ok := vm.Breakpoints[int(vm.PC)]; ok {
		return Breakpoint{Address: int(vm.PC), Reason: s}
	}

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

/// System call an RCA 1802 program at an Address.
///
func (vm *CHIP_8) sys(address uint) {
	// unimplemented
}

/// Call a subroutine at Address.
///
func (vm *CHIP_8) call(address uint) {
	if int(vm.SP) >= len(vm.Stack) {
		panic("Stack overflow!")
	}

	// post increment
	vm.Stack[vm.SP] = vm.PC
	vm.SP += 1

	// jump to Address
	vm.PC = address
}

/// Return from subroutine.
///
func (vm *CHIP_8) ret() {
	if vm.SP == 0 {
		panic("Stack underflow!")
	}

	// pre-decrement
	vm.SP -= 1
	vm.PC = vm.Stack[vm.SP]
}

/// Exit the interpreter.
///
func (vm *CHIP_8) exit() {
	vm.PC -= 2
}

/// Set low res mode.
///
func (vm *CHIP_8) low() {
	vm.Pitch = 8
}

/// Set high res mode.
///
func (vm *CHIP_8) high() {
	vm.Pitch = 16
}

/// Scroll n pixels up.
///
func (vm *CHIP_8) scrollUp(n byte) {
	if vm.Pitch == 8 {
		n >>= 1
	}

	// shift all the pixels up
	copy(vm.Video[:], vm.Video[uint(n)*vm.Pitch:])

	// wipe the bottom-most pixels
	for i := 0x400-uint(n)*vm.Pitch;i < 0x400;i++ {
		vm.Video[i] = 0
	}
}

/// Scroll n pixels down.
///
func (vm *CHIP_8) scrollDown(n byte) {
	if vm.Pitch == 8 {
		n >>= 1
	}

	// shift all the pixels down
	copy(vm.Video[uint(n)*vm.Pitch:], vm.Video[:])

	// wipe the top-most pixels
	for i := uint(0);i < uint(n)*vm.Pitch;i++ {
		vm.Video[i] = 0
	}
}

/// Scroll pixels right.
///
func (vm *CHIP_8) scrollRight() {
	shift := vm.Pitch>>2

	for i := uint(0x3FF);i >= 0;i-- {
		vm.Video[i] >>= shift

		// get the lower bits from the previous byte
		if i&(vm.Pitch-1) > 0 {
			vm.Video[i] |= vm.Video[i-1] << (8-shift)
		}
	}
}

/// Scroll pixels left.
///
func (vm *CHIP_8) scrollLeft() {
	shift := vm.Pitch>>2

	for i := uint(0);i < 0x400;i++ {
		vm.Video[i] <<= shift

		// get the upper bits from the next byte
		if i&(vm.Pitch-1) < (vm.Pitch-1) {
			vm.Video[i] |= vm.Video[i+1] >> (8-shift)
		}
	}
}

/// Jump to Address.
///
func (vm *CHIP_8) jump(address uint) {
	vm.PC = address
}

/// Jump to Address + v0.
///
func (vm *CHIP_8) jumpV0(address uint) {
	vm.PC = address + uint(vm.V[0])
}

/// Skip next instruction if vx == n.
///
func (vm *CHIP_8) skipIf(x uint, b byte) {
	if vm.V[x] == b {
		vm.PC += 2
	}
}

/// Skip next instruction if vx != n.
///
func (vm *CHIP_8) skipIfNot(x uint, b byte) {
	if vm.V[x] != b {
		vm.PC += 2
	}
}

/// Skip next instruction if vx == vy.
///
func (vm *CHIP_8) skipIfXY(x, y uint) {
	if vm.V[x] == vm.V[y] {
		vm.PC += 2
	}
}

/// Skip next instruction if vx != vy.
///
func (vm *CHIP_8) skipIfNotXY(x, y uint) {
	if vm.V[x] != vm.V[y] {
		vm.PC += 2
	}
}

/// Skip next instruction if key(vx) is pressed.
///
func (vm *CHIP_8) skipIfPressed(x uint) {
	if vm.Keys[vm.V[x]] {
		vm.PC += 2
	}
}

/// Skip next instruction if key(vx) is not pressed.
///
func (vm *CHIP_8) skipIfNotPressed(x uint) {
	if !vm.Keys[vm.V[x]] {
		vm.PC += 2
	}
}

/// Load n into vx.
///
func (vm *CHIP_8) loadX(x uint, b byte) {
	vm.V[x] = b
}

/// Load y into vx.
///
func (vm *CHIP_8) loadXY(x, y uint) {
	vm.V[x] = vm.V[y]
}

/// Load delay timer into vx.
///
func (vm *CHIP_8) loadXDT(x uint) {
	vm.V[x] = vm.GetDelayTimer()
}

/// Load vx into delay timer.
///
func (vm *CHIP_8) loadDTX(x uint) {
	vm.DT = time.Now().UnixNano() + int64(vm.V[x])*1000000000/60
}

/// Load vx into sound timer.
///
func (vm *CHIP_8) loadSTX(x uint) {
	vm.ST = time.Now().UnixNano() + int64(vm.V[x])*1000000000/60
}

/// Load vx with next key hit (blocking).
///
func (vm *CHIP_8) loadXK(x uint) {
	vm.W = &vm.V[x]
}

/// Load Address register.
///
func (vm *CHIP_8) loadI(address uint) {
	vm.I = address
}

/// Load Address with BCD of vx.
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

/// Load font sprite for vx into I.
///
func (vm *CHIP_8) loadF(x uint) {
	vm.I = uint(vm.V[x]) * 5
}

/// Load high font sprite for vx into I.
///
func (vm *CHIP_8) loadHF(x uint) {
	vm.I = 0x50 + uint(vm.V[x])*10
}

/// Bitwise or vx with vy into vx.
///
func (vm *CHIP_8) or(x, y uint) {
	vm.V[x] |= vm.V[y]
}

/// Bitwise and vx with vy into vx.
///
func (vm *CHIP_8) and(x, y uint) {
	vm.V[x] &= vm.V[y]
}

/// Bitwise xor vx with vy into vx.
///
func (vm *CHIP_8) xor(x, y uint) {
	vm.V[x] ^= vm.V[y]
}

/// Bitwise shift vx 1 bit, set carry to MSB of vx before shift.
///
func (vm *CHIP_8) shl(x uint) {
	vm.V[0xF] = vm.V[x] >> 7
	vm.V[x] <<= 1
}

/// Bitwise shift vx 1 bit, set carry to LSB of vx before shift.
///
func (vm *CHIP_8) shr(x uint) {
	vm.V[0xF] = vm.V[x] & 1
	vm.V[x] >>= 1
}

/// Add n to vx.
///
func (vm *CHIP_8) addX(x uint, b byte) {
	vm.V[x] += b
}

/// Add vy to vx and set carry.
///
func (vm *CHIP_8) addXY(x, y uint) {
	vm.V[x] += vm.V[y]

	if vm.V[x] < vm.V[y] {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}
}

/// Add v to i.
///
func (vm *CHIP_8) addIX(x uint) {
	vm.I += uint(vm.V[x])

	if vm.I >= 0x1000 {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}
}

/// Subtract vy from vx, set carry if no borrow.
///
func (vm *CHIP_8) subXY(x, y uint) {
	if vm.V[x] >= vm.V[y] {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}

	vm.V[x] -= vm.V[y]
}

/// Subtract vx from vy and store in vx, set carry if no borrow.
///
func (vm *CHIP_8) subYX(x, y uint) {
	if vm.V[y] >= vm.V[x] {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}

	vm.V[x] = vm.V[y] - vm.V[x]
}

/// Load a random number & n into vx.
///
func (vm *CHIP_8) loadRandom(x uint, b byte) {
	vm.V[x] = byte(rand.Int31() & int32(b))
}

/// Draw a sprite in memory to video at x,y with a height of n.
///
func (vm *CHIP_8) draw(a, x, y uint, n byte) byte {
	c := byte(0)

	// byte offset and bit index
	b := x>>3
	i := x&7

	// which scan line will it render on
	y = y*vm.Pitch

	// draw each row of the sprite
	for _, s := range vm.Memory[a: a + uint(n)] {
		n := y+b

		// clip pixels that are off screen
		if (n >= 256 && vm.Pitch == 8) || (n >= 1024 && vm.Pitch == 16) {
			continue
		}

		// origin pixel values
		b0 := vm.Video[n]
		b1 := vm.Video[n+1]

		// xor pixels
		vm.Video[n] ^= s >> i

		// are there pixels overlapping next byte?
		if i > 0 {
			vm.Video[n+1] ^= s << (8-i)
		}

		// were any pixels turned off?
		c |= b0 & ^vm.Video[n]
		c |= b1 & ^vm.Video[n+1]

		// next scan line
		y += vm.Pitch
	}

	// non-zero if there was a collision
	return c
}

/// Draw a sprite at I to video memory at vx, vy.
///
func (vm *CHIP_8) drawSprite(x, y uint, n byte) {
	if vm.draw(vm.I, uint(vm.V[x]), uint(vm.V[y]), n) != 0 {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}
}

/// Draw an extended 16x16 sprite at I to video memory to vx, vy.
///
func (vm *CHIP_8) drawSpriteEx(x, y uint) {
	c := byte(0)
	a := vm.I

	// draw sprite columns
	for i := uint(0);i < 16;i++ {
		c |= vm.draw(a+(i<<1), uint(vm.V[x]), uint(vm.V[y])+i, 1)

		if vm.Pitch == 16 {
			c |= vm.draw(a+(i<<1)+1, uint(vm.V[x])+8, uint(vm.V[y])+i, 1)
		}
	}

	// set the collision flag
	if c != 0 {
		vm.V[0xF] = 1
	} else {
		vm.V[0xF] = 0
	}
}

/// Save registers v0..vx to I.
///
func (vm *CHIP_8) saveRegs(x uint) {
	copy(vm.Memory[vm.I:], vm.V[:x+1])
}

/// Load registers v0..vx from I.
///
func (vm *CHIP_8) loadRegs(x uint) {
	copy(vm.V[:], vm.Memory[vm.I:vm.I+x+1])
}

/// Store v0..v7 in the HP-RPL user flags.
///
func (vm *CHIP_8) storeR(x uint) {
	copy(vm.R[:], vm.V[:x + 1])
}

/// Read the HP-RPL user flags into v0..v7.
///
func (vm *CHIP_8) readR(x uint) {
	copy(vm.V[:], vm.R[:x + 1])
}
