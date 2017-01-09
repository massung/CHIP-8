# CHIP-8 Emulator

![CHIP-8 Screenshot](data/screenshot.png "The game 'CAR'")

CHIP-8 is an assembler, debugger, and emulator for the [COSMAC ELF][2] [CHIP-8][3] interpreter and its derivative: the Super CHIP-8, which ran on HP-48 calculators. Everything is emulated as well as possible: the video display refreshes at 60 Hz and sound is emulated as well.

From the screenshot above you can see the disassembled program, register values, and a log which is used to show breakpoint information, memory dumps, and more.

CHIP-8 is written in [Go](https://golang.org/) and uses [SDL](https://www.libsdl.org/) for its rendering, input handling, and audio. It should easily run on Windows, OS X, and Linux.

## Usage

```
CHIP-8 [-b] [ROM|C8]
```

Simply pass the filename of the ROM or a .C8 assembly source file to the executable and CHIP-8 will load it, assemble if required, and begin running it. If no ROM or C8 file is specified then a default ROM ([Pong](https://en.wikipedia.org/wiki/Pong)) is loaded. 

If `-b` is passed as a command line flag, then the CHIP-8 emulator will start with a breakpoint at the first address executed.

Once the program is running, press `F1` at any time to see the list of key commands available to you. But here's a quick breakdown:

* `F1`: show help
* `[`: slow down emulation
* `]`: speed up emulation
* `BACKSPACE`: reset the emulator
* `SPACE`: pause/break emulation
* `F5..F8`: save image slot to disk 
* `Control`+`F5..F8`: load image from slot
* `F9`: toggle user breakpoint (while paused)
* `F10`: single step instruction (while paused)
* `F11`: dump memory at `I` register
* `F12`: save a screenshot to `./screenshot.bmp`

## Emulation Speed

The CHIP-8 was originally written for the [RCA CDP1802 COSMAC ELF](http://www.cosmacelf.com/), and while there is no documentation on the performance of the CHIP-8 virtual machine, after playing around with many of the games, I settled on a default of ~1,000 CHIP-8 instructions per second being emulated.

## Saved ROM Images

All disk images (`F5..F8`) are saved in `~/CHIP-8/IMG_F<5..8>`. They are a perfect snapshot of the CHIP-8 virtual machine.
 
## Assembler

While playing the games that exist for the CHIP-8 might be fun for a while, the real fun is in creating your own games and seeing just how creative you can be with such a limited machine!

Just about every assembler for the CHIP-8 is different, and this one is, too. It's designed with a few niceties in mind. So, bear this in mind and take a few minutes to peruse this documentation.

A heavily documented, example program for the game [Snake](https://en.wikipedia.org/wiki/Snake_(video_game)) can be found in [games/sources/snake.c8](https://github.com/massung/chip-8/blob/master/games/sources/snake.c8).

### Syntax

Each line of assembly uses the following syntax:

```
.label    instruction  arg0, arg1, ...   ; comment
```

A label **must** appear at the very beginning of the line, and there **must** be at least a single whitespace character before the instruction or directive of a line (i.e. an instruction cannot appear at the beginning of a line).

### Registers and Literals

The CHIP-8 has 16, 8-bit virtual registers: `V0`, `V1`, `V2`, `V3`, `V4`, `V5`, `V6`, `V7`, `V8`, `V9`, `VA`, `VB`, `VC`, `VD`, `VE`, and `VF`. All of these are considered general purpose registers except for `VF` which is used for carry, borrow, shift, overflow, and collision detection.
 
There is a single, 16-bit address register: `I`, which is used for reading from - and writing to - memory.

Last, there are two 8-bit timer registers (`DT` for delays and `ST` for sounds) that continuously count down at 60 Hz. The delay timer is good for time limiting your game and as long as the sound timer is non-zero a tone will be emitted.

There are two literal types understood by the assembler: numbers and text strings. The bases for numbers accepted are 10, 16 (`#FF`), and 2 (`$10`). Base 2 (binary) is a little special in that - since it is often used for creating sprite data - a `.` can be used instead of `0`. For example:
  
```
    LD   VC, 10   ; VC = 10
    ADD  V3, #FE  ; V3 = V3 - 2
    
    ; draw the ball sprite
    LD   I, ball
    DRW  V3, VC, 6
    
.ball
    BYTE $..1111..
    BYTE $.1....1.
    BYTE $1......1
    BYTE $1......1
    BYTE $.1....1.
    BYTE $..1111..
```

Text literals can be added with single or double quotes, but there is no escape character. Usually this is just to add text information to the final ROM and not for any game data.

```
    BYTE "A little game made by ME!"
```

### Instruction Set

Here is the CHIP-8 instructions. The Super CHIP-8 instructions follow after the basic instruction set.

| Opcode | Mnemonic      | Description
|:-------|:--------------|:---------------------------------------------------------------
| 00E0   | CLS           | Clear video memory
| 00EE   | RET           | Return from subroutine
| 0NNN   | SYS NNN       | Call CDP1802 subroutine at NNN
| 1NNN   | CALL NNN      | Call CHIP-8 subroutine at NNN
| 2NNN   | JP NNN        | Jump to address NNN
| BNNN   | JP V0, NNN    | Jump to address NNN + V0
| 3XNN   | SE VX, NN     | Skip next instruction if VX == NN
| 4XNN   | SNE VX, NN    | Skip next instruction if VX != NN
| 5XY0   | SE VX, VY     | Skip next instruction if VX == VY
| 9XY0   | SNE VX, VY    | Skip next instruction if VX != VY
| EX9E   | SKP VX        | Skip next instruction if key(VX) is pressed
| EXA1   | SKNP VX       | Skip next instruction if key(VX) is not pressed
| FX0A   | LD VX, K      | Wait for key press, store key pressed in VX
| 6XNN   | LD VX, NN     | VX = NN
| 8XY0   | LD VX, VY     | VX = VY
| FX07   | LD VX, DT     | VX = DT
| FX15   | LD DT, VX     | DT = VX
| FX18   | LD ST, VX     | ST = VX
| ANNN   | LD I, NNN     | I = NNN
| FX29   | LD F, VX      | I = address of 4x5 font character in VX (0..F)
| FX33   | LD B, VX      | Store BCD representation of VX at I (100), I+1 (10), and I+2 (1)
| FX55   | LD [I], VX    | Store V0..VX (inclusive) to memory starting at I
| FX65   | LD VX, [I]    | Load V0..VX (inclusive) from memory starting at I
| FX1E   | ADD I, VX     | I = I + VX; VF = if I > 0xFFF then 1 else 0
| 7XNN   | ADD VX, NN    | VX = VX + NN
| 8XY4   | ADD VX, VY    | VX = VX + VY; VF = if carry then 1 else 0
| 8XY5   | SUB VX, VY    | VX = VX - VY; VF = if borrow then 0 else 1
| 8XY7   | SUBN VX, VY   | VX = VY - VX; VF = if borrow then 0 else 1
| 8XY1   | OR VX, VY     | VX = VX OR VY
| 8XY2   | AND VX, VY    | VX = VX AND VY
| 8XY3   | XOR VX, VY    | VX = VX XOR VY
| 8XY6   | SHR VX        | VF = LSB(VX); VX = VX >> 1
| 8XYE   | SHL VX        | VF = MSB(VX); VX = VX << 1
| CXNN   | RND VX, NN    | VX = RND() AND NN
| DXYN   | DRW VX, VY, N | Draw 8xN sprite at I to VX, VY; VF = if collision then 1 else 0

And here are the instructions added for the Super CHIP-8 (a.k.a. CHIP-48):

| Opcode | Mnemonic      | Description
|:-------|:--------------|:---------------------------------------------------------------
| 00BN   | SCU N         | Scroll up N pixels (N/2 pixels in low res mode)
| 00CN   | SCD N         | Scroll down N pixels (N/2 pixels in low res mode)
| 00FB   | SCR           | Scroll right 4 pixels (2 pixels in low res mode)
| 00FC   | SCL           | Scroll left 4 pixels (2 pixels in low res mode)
| 00FD   | EXIT          | Exit the interpreter; this causes the VM to infinite loop
| 00FE   | LOW           | Enter low resolution (64x32) mode; this is the default mode
| 00FF   | HIGH          | Enter high resolution (128x64) mode
| DXY0   | DRW VX, VY, 0 | Draw a 16x16 sprite at I to VX, VY (8x16 in low res mode)
| FX30   | LD HF, VX     | I = address of 8x10 font character in VX (0..F)
| FX75   | LD R, VX      | Store V0..VX (inclusive) into HP-RPL user flags (X < 8)
| FX85   | LD VX, R      | Load V0..VX (inclusive) from HP-RPL user flags (X < 8)

_NOTE: Nothing special needs to be done to use the Super CHIP-8 instructions. They are just noted separately for anyone wishing to hack the code, so they are aware that they are not part of the original CHIP-8 virtual machine._

### Directives

The assembler understands - beyond instructions - the following directives:

* `DECLARE` .. `AS`
* `BREAK`
* `ASSERT`
* `BYTE`
* `WORD`
* `ALIGN`
* `RESERVE`

Use `DECLARE` .. `AS` to declare a global identifier for use in lieu of a literal constant, register, or label. Very handy when using specific registers as global variables to make the code more clear and easy to refactor. _Unlike labels, declares must happen **before** being used._

```
    declare score as v5
```

Use `BREAK` to create a breakpoint in code. Nothing is written to the ROM, but when the next instruction is reached, the emulator will pause and allow you to single-step the code and/or inspect memory, registers, etc. Any text following the `BREAK` will be visible in the log upon hitting the breakpoint.

```
    break   check player death
```

Use `ASSERT` to create a conditional breakpoint. It will only trigger if `VF` is _non-zero_ (i.e. it can be used for conditions other than carry, borrow, address overflow, and collision detection) when hit.

```
    assert  score overflowed!
```

Use `BYTE` to write successive bytes to the ROM. This can take one or more byte literals or strings as arguments.

```
    byte    1, #FF, $1001, $1..1, "Hello, world!"
```

Use `WORD` to write successive 2-byte words to the ROM. Remember that all words are stored with big-endian (MSB first) byte ordering.

```
    word    1, #FFFF
```

Use `ALIGN` to align the ROM to a specific byte boundary. The boundary **must** be a power of 2.

```
    align   32
```

Use `RESERVE` to simply write N successive zeros to the ROM in order to reserve memory. Technically no different than `BYTE 0, 0, 0, 0` just easier and more obvious.

```
    reserve 256
```

### CHIP-8 Tips & Tricks

Assembly language - if you're not used to it - can be a bit daunting at first. Here's some tips to keep in mind (for CHIP-8 and assembly programming in general) that can help you along the way...

* If you want to subtract a constant value from a register, remember it's easier to just add the [two's complement](https://en.wikipedia.org/wiki/Two%27s_complement) instead.

* Want to compare greater than? Use `SUB` and `SUBN`. Remember `VF` is 1 if there is **not** a borrow (read: the result is >= 0). Use `SUBN` when you want to compare, but not store the result in what you're comparing.

* Need a switch statement? Use `SE` and `SNE` followed by `JP` instructions to build a jump table.

* Use a `JP V0, label` instead of `SE` switches. This isn't always possible, but better when it is.

* Perform [tail calls](https://en.wikipedia.org/wiki/Tail_call) whenever possible. If you see a `CALL` followed by a `RET`, just change the `CALL` to a `JP` and get rid of the `RET`.

* Need a random point on the screen? `RND VX, #3F` for X and `RND VY, #1F` for Y. Use `#7F` and `#3F` if in high res mode.

* When setting up global use of registers, leave `V0`-`V2` always free as scratch. They are incredibly useful for loading from `LD V2, [I]`, especially after performing a BCD conversion.

* Have some tips? Email them and I'll be sure to add them... Once there's enough, it might be useful to make a whole page just about that!

### Examples

There are a few example programs in `games/sources` for you you play around with, modify, and learn from.

## That's all folks!

If you have any feedback, please email me. If you find a bug or would like a feature, feel free to open an issue. 

[1]: http://www.cosmacelf.com/
[2]: https://en.wikipedia.org/wiki/RCA_1802
[3]: https://en.wikipedia.org/wiki/CHIP-8/
