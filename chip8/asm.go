/* Copyright (c) 2017 Jeffrey Massung
 *
 * This software is provided 'as-is', without any express or implied
 * warranty.  In no event will the authors be held liable for any damages
 * arising from the use of this software.
 *
 * Permission is granted to anyone to use this software for any purpose,
 * including commercial applications, and to alter it and redistribute it
 * freely, subject to the following restrictions:
 *
 * 1. The origin of this software must not be misrepresented; you must not
 *    claim that you wrote the original software. If you use this software
 *    in a product, an acknowledgment in the product documentation would be
 *    appreciated but is not required.
 *
 * 2. Altered source versions must be plainly marked as such, and must not be
 *    misrepresented as being the original software.
 *
 * 3. This notice may not be removed or altered from any source distribution.
 */

package chip8

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

/// Assembly is a completely assembled source file.
///
type Assembly struct {
	/// ROM is the final, assembled bytes to load.
	///
	ROM []byte

	/// Breakpoints is a list of addresses.
	///
	Breakpoints []Breakpoint

	/// Label mapping.
	///
	Labels map[string]token

	/// Addresses with unresolved labels.
	///
	Unresolved map[int]string

	/// Base address the ROM begins at (0x200 or 0x600 for ETI).
	///
	Base int

	/// Super is true if using additional super CHIP-8 instructions.
	///
	Super bool

	/// Extended is true if using additional CHIP-8E instructions.
	///
	Extended bool
}

var (
	/// AsciiTable is the 6-bit ASCII table for CHIP-8E.
	///
	AsciiTable = `@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\]^_ !"#$%&'()*+,-./0123456789:;<=>?`
)

/// Assemble an input CHIP-8 source code file.
///
func Assemble(program []byte, eti bool) (out *Assembly, err error) {
	var line int

	// base address for program
	base := 0x200

	// ETI-660 binaries are loaded to 0x600
	if eti {
		base = 0x600
	}

	// create an empty, return assembly
	out = &Assembly{
		ROM: make([]byte, base, 0x1000),
		Breakpoints: make([]Breakpoint, 0, 10),
		Labels: make(map[string]token),
		Unresolved: make(map[int]string),
		Base: base,
	}

	// no error
	err = nil

	// handle panics during assembly
	defer func() {
		if r := recover(); r != nil {
			if line > 0 {
				err = fmt.Errorf("line %d - %s", line, r)
			} else {
				err = fmt.Errorf("%s", r)
			}

			// return a dummy ROM
			out = &Assembly{ROM: Dummy}
		}
	}()

	// create simple line scanner over the file
	reader := bytes.NewReader(bytes.ToUpper(program))
	scanner := bufio.NewScanner(reader)

	// parse and assemble
	for line = 1;scanner.Scan();line++ {
		out.assemble(&tokenScanner{bytes: scanner.Bytes()})
	}

	// resolve all label addresses
	for address, label := range out.Unresolved {
		if t, ok := out.Labels[label]; ok {
			if t.typ != TOKEN_LIT {
				panic("label does not resolve to address!")
			}

			msb := byte(t.val.(int)>>8)
			lsb := byte(t.val.(int)&0xFF)

			// NOTE: This "just works" because all labels are guaranteed to be
			//       addressed within 12-bits. There are only a handful of
			//       instructions that take an immediate Address:
			//
			//         SYS    NNN
			//         CALL   NNN
			//         JP     NNN
			//         JP     V0, NNN
			//         LD     I, NNN
			//
			//       The only other use case is the WORD instruction to write
			//       16-bit values, and since the unresolved label defaulted
			//       to 0x0200, overwriting it works just fine.
			//
			out.ROM[address] = msb | (out.ROM[address]&0xF0)
			out.ROM[address+1] = lsb

			// delete the unresolved Address
			delete(out.Unresolved, address)
		}
	}

	// clear the line number as we're done assembling
	line = 0

	// if there are any unresolved addresses, panic
	for _, label := range out.Unresolved {
		panic(fmt.Errorf("unresolved label: %s", label))
	}

	// drop the first 512 bytes from the rom
	out.ROM = out.ROM[base:]

	// done
	return
}

/// Compile a single line into the assembly.
///
func (a *Assembly) assemble(s *tokenScanner) {
	t := s.scanToken()

	// assign labels
	if t.typ == TOKEN_LABEL {
		t = a.assembleLabel(t.val.(string), s)
	}

	// continue assembling
	switch {
	case t.typ == TOKEN_INSTRUCTION:
		a.assembleInstruction(t.val.(string), s)
	case t.typ == TOKEN_SUPER:
		a.assembleSuper(s)
	case t.typ == TOKEN_EXTENDED:
		a.assembleExtended(s)
	case t.typ == TOKEN_BREAK:
		a.assembleBreakpoint(s, false)
	case t.typ == TOKEN_ASSERT:
		a.assembleBreakpoint(s, true)
	case t.typ != TOKEN_END:
		panic("unexpected token")
	}

}

/// Scan for a label and add it to the assembly.
///
func (a *Assembly) assembleLabel(label string, s *tokenScanner) token {
	if _, exists := a.Labels[label]; exists {
		panic("duplicate label")
	}

	// by default, the label is assigned the current address
	a.Labels[label] = token{typ: TOKEN_LIT, val: len(a.ROM)}

	// scan the next token
	t := s.scanToken()

	// if EQU or VAR, reassign the label
	if t.typ == TOKEN_EQU || t.typ == TOKEN_VAR {
		v := s.scanToken()

		// equ requires a literal, and var requires a v-register
		if (t.typ == TOKEN_EQU && v.typ == TOKEN_LIT) || (t.typ == TOKEN_VAR && v.typ == TOKEN_V) {
			a.Labels[label] = v

			// should be the final token
			if t = s.scanToken(); t.typ == TOKEN_END {
				return t
			}
		}

		panic("illegal label assignment")
	}

	return t
}

/// Create a new breakpoint at the current Address.
///
func (a *Assembly) assembleBreakpoint(s *tokenScanner, conditional bool) {
	reason := s.scanToEnd().val.(string)

	// create the breakpoint
	a.Breakpoints = append(a.Breakpoints, Breakpoint{
		Address: len(a.ROM),
		Conditional: conditional,
		Reason: reason,
	})
}

/// Allow the assembler to assemble super, SCHIP-8 instructions.
///
func (a *Assembly) assembleSuper(s *tokenScanner) {
	if s.scanToken().typ != TOKEN_END {
		panic("unexpected token")
	}

	if len(a.ROM) > a.Base {
		panic("super must come before instructions")
	}

	// enter super instructions mode
	a.Super = true
}

/// Allow the assembler to assemble extended, CHIP-8E instructions.
///
func (a *Assembly) assembleExtended(s *tokenScanner) {
	if s.scanToken().typ != TOKEN_END {
		panic("unexpected token")
	}

	if len(a.ROM) > a.Base {
		panic("extended must come before instructions")
	}

	// enter extended instructions mode
	a.Extended = true
}

/// Compile a single instruction into the assembly.
///
func (a *Assembly) assembleInstruction(i string, s *tokenScanner) {
	tokens := s.scanOperands()

	switch i {
	case "CLS":
		a.ROM = append(a.ROM, a.assembleCLS(tokens)...)
	case "RET":
		a.ROM = append(a.ROM, a.assembleRET(tokens)...)
	case "EXIT":
		a.ROM = append(a.ROM, a.assembleEXIT(tokens)...)
	case "LOW":
		a.ROM = append(a.ROM, a.assembleLOW(tokens)...)
	case "HIGH":
		a.ROM = append(a.ROM, a.assembleHIGH(tokens)...)
	case "SCU":
		a.ROM = append(a.ROM, a.assembleSCU(tokens)...)
	case "SCD":
		a.ROM = append(a.ROM, a.assembleSCD(tokens)...)
	case "SCR":
		a.ROM = append(a.ROM, a.assembleSCR(tokens)...)
	case "SCL":
		a.ROM = append(a.ROM, a.assembleSCL(tokens)...)
	case "SYS":
		a.ROM = append(a.ROM, a.assembleSYS(tokens)...)
	case "JP":
		a.ROM = append(a.ROM, a.assembleJP(tokens)...)
	case "CALL":
		a.ROM = append(a.ROM, a.assembleCALL(tokens)...)
	case "SE":
		a.ROM = append(a.ROM, a.assembleSE(tokens)...)
	case "SNE":
		a.ROM = append(a.ROM, a.assembleSNE(tokens)...)
	case "SGT":
		a.ROM = append(a.ROM, a.assembleSGT(tokens)...)
	case "SLT":
		a.ROM = append(a.ROM, a.assembleSLT(tokens)...)
	case "SKP":
		a.ROM = append(a.ROM, a.assembleSKP(tokens)...)
	case "SKNP":
		a.ROM = append(a.ROM, a.assembleSKNP(tokens)...)
	case "OR":
		a.ROM = append(a.ROM, a.assembleOR(tokens)...)
	case "AND":
		a.ROM = append(a.ROM, a.assembleAND(tokens)...)
	case "XOR":
		a.ROM = append(a.ROM, a.assembleXOR(tokens)...)
	case "SHR":
		a.ROM = append(a.ROM, a.assembleSHR(tokens)...)
	case "SHL":
		a.ROM = append(a.ROM, a.assembleSHL(tokens)...)
	case "ADD":
		a.ROM = append(a.ROM, a.assembleADD(tokens)...)
	case "SUB":
		a.ROM = append(a.ROM, a.assembleSUB(tokens)...)
	case "SUBN":
		a.ROM = append(a.ROM, a.assembleSUBN(tokens)...)
	case "MUL":
		a.ROM = append(a.ROM, a.assembleMUL(tokens)...)
	case "DIV":
		a.ROM = append(a.ROM, a.assembleDIV(tokens)...)
	case "BCD":
		a.ROM = append(a.ROM, a.assembleBCD(tokens)...)
	case "RND":
		a.ROM = append(a.ROM, a.assembleRND(tokens)...)
	case "DRW":
		a.ROM = append(a.ROM, a.assembleDRW(tokens)...)
	case "LD":
		a.ROM = append(a.ROM, a.assembleLD(tokens)...)
	case "ASCII":
		a.ROM = append(a.ROM, a.assembleASCII(tokens)...)
	case "BYTE":
		a.ROM = append(a.ROM, a.assembleBYTE(tokens)...)
	case "WORD":
		a.ROM = append(a.ROM, a.assembleWORD(tokens)...)
	case "ALIGN":
		a.ROM = append(a.ROM, a.assembleALIGN(tokens)...)
	case "PAD":
		a.ROM = append(a.ROM, a.assemblePAD(tokens)...)
	}
}

/// Assemble a single operand, expanding label references.
///
func (a *Assembly) assembleOperand(t token) token {
	if t.typ == TOKEN_ID {
		label := t.val.(string)
		if v, exists := a.Labels[label]; exists {
			t = v
		} else {
			t = token{typ: TOKEN_LIT, val: 0x200}

			// add an unresolved address
			a.Unresolved[len(a.ROM)] = label
		}
	}

	return t
}

/// Match the desired tokens with a list of tokens. Expand defines and labels.
///
func (a *Assembly) assembleOperands(tokens []token, m ...tokenType) ([]token, bool) {
	ops := make([]token, 0, 3)

	// the number of desired tokens should match
	if len(tokens) != len(m) {
		return nil, false
	}

	// expand and compare the token types
	for i, typ := range m {
		t := a.assembleOperand(tokens[i])

		// compare token types
		if t.typ != typ {
			return nil, false
		}

		// append the operand
		ops = append(ops, t)
	}

	return ops, true
}

/// Assemble a CLS instruction.
///
func (a *Assembly) assembleCLS(tokens []token) []byte {
	if len(tokens) == 0 {
		return []byte{0x00, 0xE0}
	}

	panic("illegal instruction")
}

/// Assemble a RET instruction.
///
func (a *Assembly) assembleRET(tokens []token) []byte {
	if len(tokens) == 0 {
		return []byte{0x00, 0xEE}
	}

	panic("illegal instruction")
}

/// Assemble an EXIT instruction.
///
func (a *Assembly) assembleEXIT(tokens []token) []byte {
	if a.Super {
		if len(tokens) == 0 {
			return []byte{0x00, 0xFD}
		}
	}

	panic("illegal instruction")
}

/// Assemble a LOW instruction.
///
func (a *Assembly) assembleLOW(tokens []token) []byte {
	if a.Super {
		if len(tokens) == 0 {
			return []byte{0x00, 0xFE}
		}
	}

	panic("illegal instruction")
}

/// Assemble a HIGH instruction.
///
func (a *Assembly) assembleHIGH(tokens []token) []byte {
	if a.Super {
		if len(tokens) == 0 {
			return []byte{0x00, 0xFF}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SCU instruction.
///
func (a *Assembly) assembleSCU(tokens []token) []byte {
	if a.Super {
		if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
			n := ops[0].val.(int)

			if n < 0x10 {
				return []byte{0x00, 0xB0 | byte(n)}
			}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SCD instruction.
///
func (a *Assembly) assembleSCD(tokens []token) []byte {
	if a.Super {
		if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
			n := ops[0].val.(int)

			if n < 0x10 {
				return []byte{0x00, 0xC0 | byte(n)}
			}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SCR instruction.
///
func (a *Assembly) assembleSCR(tokens []token) []byte {
	if a.Super {
		if len(tokens) == 0 {
			return []byte{0x00, 0xFB}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SCL instruction.
///
func (a *Assembly) assembleSCL(tokens []token) []byte {
	if a.Super {
		if len(tokens) == 0 {
			return []byte{0x00, 0xFC}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SYS instruction.
///
func (a *Assembly) assembleSYS(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		a := ops[0].val.(int)

		if a < 0x1000 {
			return []byte{byte(a >> 8 & 0xF), byte(a & 0xFF)}
		}
	}

	panic("illegal instruction")
}

/// Assemble a JP instruction.
///
func (a *Assembly) assembleJP(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		a := ops[0].val.(int)

		if a < 0x1000 {
			return []byte{0x10|byte(a >> 8 & 0xF), byte(a & 0xFF)}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_LIT); ok {
		v := ops[0].val.(int)
		a := ops[1].val.(int)

		if v == 0 && a < 0x1000 {
			return []byte{0xB0|byte(a >> 8 & 0xF), byte(a & 0xFF)}
		}
	}

	panic("illegal instruction")
}

/// Assemble a CALL instruction.
///
func (a *Assembly) assembleCALL(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		a := ops[0].val.(int)

		if a < 0x1000 {
			return []byte{0x20|byte(a >> 8 & 0xF), byte(a & 0xFF)}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SE instruction.
///
func (a *Assembly) assembleSE(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return []byte{0x30|byte(x), byte(b)}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x50|byte(x), byte(y << 4)}
	}

	panic("illegal instruction")
}

/// Assemble a SNE instruction.
///
func (a *Assembly) assembleSNE(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return []byte{0x40|byte(x), byte(b)}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x90|byte(x), byte(y << 4)}
	}

	panic("illegal instruction")
}

/// Assemble a SGT instruction.
///
func (a *Assembly) assembleSGT(tokens []token) []byte {
	if a.Extended {
		if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
			x := ops[0].val.(int)
			y := ops[1].val.(int)

			return []byte{0x50 | byte(x), byte(y << 4) | 0x01}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SLT instruction.
///
func (a *Assembly) assembleSLT(tokens []token) []byte {
	if a.Extended {
		if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
			x := ops[0].val.(int)
			y := ops[1].val.(int)

			return []byte{0x50 | byte(x), byte(y << 4) | 0x02}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SKP instruction.
///
func (a *Assembly) assembleSKP(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0xE0|byte(x), 0x9E}
	}

	panic("illegal instruction")
}

/// Assemble a SKNP instruction.
///
func (a *Assembly) assembleSKNP(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0xE0|byte(x), 0xA1}
	}

	panic("illegal instruction")
}

/// Assemble a OR instruction.
///
func (a *Assembly) assembleOR(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x01}
	}

	panic("illegal instruction")
}

/// Assemble a AND instruction.
///
func (a *Assembly) assembleAND(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x02}
	}

	panic("illegal instruction")
}

/// Assemble a XOR instruction.
///
func (a *Assembly) assembleXOR(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x03}
	}

	panic("illegal instruction")
}

/// Assemble a SHR instruction.
///
func (a *Assembly) assembleSHR(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0x80|byte(x), byte(x << 4) | 0x06}
	}

	panic("illegal instruction")
}

/// Assemble a SHL instruction.
///
func (a *Assembly) assembleSHL(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0x80|byte(x), byte(x << 4) | 0x0E}
	}

	panic("illegal instruction")
}

/// Assemble a ADD instruction.
///
func (a *Assembly) assembleADD(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return []byte{0x70|byte(x), byte(b)}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x04}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_I, TOKEN_V); ok {
		x := ops[1].val.(int)

		return []byte{0xF0|byte(x), 0x1E}
	}

	panic("illegal instruction")
}

/// Assemble a SUB instruction.
///
func (a *Assembly) assembleSUB(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x05}
	}

	panic("illegal instruction")
}

/// Assemble a SUBN instruction.
///
func (a *Assembly) assembleSUBN(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x07}
	}

	panic("illegal instruction")
}

/// Assemble a MUL instruction.
///
func (a *Assembly) assembleMUL(tokens []token) []byte {
	if a.Extended {
		if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
			x := ops[0].val.(int)
			y := ops[1].val.(int)

			return []byte{0x90 | byte(x), byte(y << 4) | 0x01}
		}
	}

	panic("illegal instruction")
}

/// Assemble a DIV instruction.
///
func (a *Assembly) assembleDIV(tokens []token) []byte {
	if a.Extended {
		if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
			x := ops[0].val.(int)
			y := ops[1].val.(int)

			return []byte{0x90 | byte(x), byte(y << 4) | 0x02}
		}
	}

	panic("illegal instruction")
}

/// Assemble a BCD instruction.
///
func (a *Assembly) assembleBCD(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0xF0|byte(x), 0x33}
	}

	if a.Extended {
		if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
			x := ops[0].val.(int)
			y := ops[1].val.(int)

			return []byte{0x90|byte(x), byte(y << 4)|0x03}
		}
	}

	panic("illegal instruction")
}

/// Assemble a RND instruction.
///
func (a *Assembly) assembleRND(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return []byte{0xC0|byte(x), byte(b)}
		}
	}

	panic("illegal instruction")
}

/// Assemble a DRW instruction.
///
func (a *Assembly) assembleDRW(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)
		n := ops[2].val.(int)

		if n < 0x10 {
			return []byte{0xD0|byte(x), byte(y << 4) | byte(n)}
		}
	}

	panic("illegal instruction")
}

/// Assemble a LD instruction.
///
func (a *Assembly) assembleLD(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return []byte{0x60|byte(x), byte(b)}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4)}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_I, TOKEN_LIT); ok {
		a := ops[1].val.(int)

		if a < 0x1000 {
			return []byte{0xA0|byte(a >> 8 & 0xF), byte(a & 0xFF)}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_DT); ok {
		x := ops[0].val.(int)

		return []byte{0xF0|byte(x), 0x07}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_K); ok {
		x := ops[0].val.(int)

		return []byte{0xF0|byte(x), 0x0A}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_DT, TOKEN_V); ok {
		x := ops[1].val.(int)

		return []byte{0xF0|byte(x), 0x15}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_ST, TOKEN_V); ok {
		x := ops[1].val.(int)

		return []byte{0xF0|byte(x), 0x18}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_F, TOKEN_V); ok {
		x := ops[1].val.(int)

		return []byte{0xF0|byte(x), 0x29}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_EFFECTIVE_ADDRESS, TOKEN_V); ok {
		x := ops[1].val.(int)

		return []byte{0xF0|byte(x), 0x55}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_EFFECTIVE_ADDRESS); ok {
		x := ops[0].val.(int)

		return []byte{0xF0|byte(x), 0x65}
	}

	if a.Super {
		if ops, ok := a.assembleOperands(tokens, TOKEN_HF, TOKEN_V); ok {
			x := ops[1].val.(int)

			return []byte{0xF0 | byte(x), 0x30}
		}

		if ops, ok := a.assembleOperands(tokens, TOKEN_R, TOKEN_V); ok {
			x := ops[1].val.(int)

			if x < 8 {
				return []byte{0xF0 | byte(x), 0x75}
			}
		}

		if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_R); ok {
			x := ops[0].val.(int)

			if x < 8 {
				return []byte{0xF0 | byte(x), 0x85}
			}
		}
	}

	if a.Extended {
		if ops, ok := a.assembleOperands(tokens, TOKEN_ASCII, TOKEN_V); ok {
			x := ops[1].val.(int)

			return []byte{0xF0 | byte(x), 0x94}
		}
	}

	panic("illegal instruction")
}

/// Assemble an ASCII instruction.
///
func (a *Assembly) assembleASCII(tokens []token) []byte {
	var b []byte

	if !a.Extended {
		panic("illegal directive")
	}

	// loop over all string tokens and assemble them as 6-bit ascii
	for _, t := range tokens {
		op := a.assembleOperand(t)

		if op.typ != TOKEN_TEXT {
			panic("expected ascii string")
		}

		// loop over each byte in the string, write the ascii table value
		for _, c := range op.val.(string) {
			if i := strings.IndexRune(AsciiTable, c); i < 0 {
				panic("invalid CHIP-8E ascii character")
			} else {
				b = append(b, byte(i))
			}
		}
	}

	return b
}

/// Assemble a BYTE instruction.
///
func (a *Assembly) assembleBYTE(tokens []token) []byte {
	b := make([]byte, 0)

	for _, t := range tokens {
		op := a.assembleOperand(t)

		switch op.typ {
		case TOKEN_LIT:
			if op.val.(int) > 0xFF {
				panic("invalid byte")
			}

			b = append(b, byte(t.val.(int)))
		case TOKEN_TEXT:
			b = append(b, op.val.(string)...)
		}

	}

	return b
}

/// Assemble a WORD instruction.
///
func (a *Assembly) assembleWORD(tokens []token) []byte {
	b := make([]byte, 0)

	for _, t := range tokens {
		op := a.assembleOperand(t)

		if op.typ != TOKEN_LIT || op.val.(int) > 0xFFFF {
			panic("invalid word")
		}

		msb := op.val.(int) >> 8 & 0xFF
		lsb := op.val.(int) & 0xFF

		// store msb first
		b = append(b, byte(msb), byte(lsb))
	}

	return b
}

/// Assemble an ALIGN instruction.
///
func (a *Assembly) assembleALIGN(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		n := ops[0].val.(int)

		if n&(n-1) == 0 {
			offset := len(a.ROM)&(n-1)
			pad := n-offset

			// reserve pad bytes to meet alignment
			return make([]byte, pad)
		}
	}

	panic("illegal alignment")
}

/// Assemble an PAD instruction.
///
func (a *Assembly) assemblePAD(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		n := ops[0].val.(int)

		if n < 0x1000 - len(a.ROM) {
			return make([]byte, n)
		}
	}

	panic("illegal size")
}
