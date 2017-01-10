package chip8

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
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

	/// Labels to addresses mapping.
	///
	Labels map[string]int

	/// Declares text substitution macros.
	///
	Declares map[string]token

	/// Addresses with unresolved labels.
	///
	Unresolved map[int]string
}

/// Assemble an input CHIP-8 source code file.
///
func Assemble(file string) (out *Assembly, err error) {
	var line int

	// create an empty, return assembly
	out = &Assembly{
		ROM: make([]byte, 0x200, 0x1000),
		Breakpoints: make([]Breakpoint, 0, 10),
		Labels: make(map[string]int),
		Declares: make(map[string]token),
		Unresolved: make(map[int]string),
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

	// read the entire file in
	content, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	// create simple line scanner over the file
	reader := bytes.NewReader(bytes.ToUpper(content))
	scanner := bufio.NewScanner(reader)

	// parse and assemble
	for line = 1;scanner.Scan();line++ {
		out.assemble(&tokenScanner{bytes: scanner.Bytes()})
	}

	// resolve all labels
	for address, label := range out.Unresolved {
		if resolved, ok := out.Labels[label]; ok {
			msb := byte(resolved>>8)
			lsb := byte(resolved&0xFF)

			// note: This "just works" because all labels are guaranteed to be
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
	out.ROM = out.ROM[0x200:]

	// done
	return
}

/// Compile a single line into the assembly.
///
func (a *Assembly) assemble(s *tokenScanner) {
	t := s.scanToken()

	switch {
	case t.typ == TOKEN_LABEL:
		a.assembleLabel(t.val.(string))
	case t.typ == TOKEN_INSTRUCTION:
		a.assembleInstruction(t.val.(string), s)
	case t.typ == TOKEN_BREAK:
		a.assembleBreakpoint(s, false)
	case t.typ == TOKEN_ASSERT:
		a.assembleBreakpoint(s, true)
	case t.typ == TOKEN_REF:
		a.assembleDeclare(t.val.(string), s)
	case t.typ != TOKEN_END:
		panic("unexpected token")
	}
}

/// Scan for a label and add it to the assembly.
///
func (a *Assembly) assembleLabel(label string) {
	if _, exists := a.Declares[label]; exists {
		panic("label exists as declare")
	}
	if _, exists := a.Labels[label]; exists {
		panic("duplicate label")
	}

	a.Labels[label] = len(a.ROM)
}

/// Create a new breakpoint at the current Address.
///
func (a *Assembly) assembleBreakpoint(s *tokenScanner, conditional bool) {
	a.Breakpoints = append(a.Breakpoints, Breakpoint{
		Address: len(a.ROM),
		Conditional: conditional,
		Reason: s.scanToEnd().val.(string),
	})
}

/// Create a new EQU identifier.
///
func (a *Assembly) assembleDeclare(id string, s *tokenScanner) {
	if _, exists := a.Declares[id]; exists {
		panic("declaration already exists")
	}
	if _, exists := a.Labels[id]; exists {
		panic("declaration already exists as label")
	}

	// scan for EQU <value>
	if t := s.scanToken(); t.typ == TOKEN_EQU {
		switch equ := s.scanToken(); equ.typ {
		case TOKEN_LIT, TOKEN_LABEL, TOKEN_V, TOKEN_I, TOKEN_F, TOKEN_HF, TOKEN_B, TOKEN_R:
			a.Declares[id] = equ

			// successfully declared
			return
		}
	}

	panic("illegal declaration")
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
	case "RND":
		a.ROM = append(a.ROM, a.assembleRND(tokens)...)
	case "DRW":
		a.ROM = append(a.ROM, a.assembleDRW(tokens)...)
	case "LD":
		a.ROM = append(a.ROM, a.assembleLD(tokens)...)
	case "BYTE":
		a.ROM = append(a.ROM, a.assembleBYTE(tokens)...)
	case "WORD":
		a.ROM = append(a.ROM, a.assembleWORD(tokens)...)
	case "ALIGN":
		a.ROM = append(a.ROM, a.assembleALIGN(tokens)...)
	case "RESERVE":
		a.ROM = append(a.ROM, a.assembleRESERVE(tokens)...)
	}
}

/// Assemble a single operand, expanding references.
///
func (a *Assembly) assembleOperand(t token) token {
	if t.typ == TOKEN_REF {
		if def, ok := a.Declares[t.val.(string)]; ok {
			return a.assembleOperand(def)
		}
	}

	// Address labels
	if t.typ == TOKEN_REF {
		if address, ok := a.Labels[t.val.(string)]; ok {
			return token{typ: TOKEN_LIT, val: address}
		}

		// add an unresolved label at this Address
		a.Unresolved[len(a.ROM)] = t.val.(string)

		// use a null label Address that's larger than a byte
		return token{typ: TOKEN_LIT, val: 0x200}
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
	if len(tokens) == 0 {
		return []byte{0x00, 0xFD}
	}

	panic("illegal instruction")
}

/// Assemble a LOW instruction.
///
func (a *Assembly) assembleLOW(tokens []token) []byte {
	if len(tokens) == 0 {
		return []byte{0x00, 0xFE}
	}

	panic("illegal instruction")
}

/// Assemble a HIGH instruction.
///
func (a *Assembly) assembleHIGH(tokens []token) []byte {
	if len(tokens) == 0 {
		return []byte{0x00, 0xFF}
	}

	panic("illegal instruction")
}

/// Assemble a SCU instruction.
///
func (a *Assembly) assembleSCU(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		n := ops[0].val.(int)

		if n < 0x10 {
			return []byte{0x00, 0xB0 | byte(n)}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SCD instruction.
///
func (a *Assembly) assembleSCD(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		n := ops[0].val.(int)

		if n < 0x10 {
			return []byte{0x00, 0xC0 | byte(n)}
		}
	}

	panic("illegal instruction")
}

/// Assemble a SCR instruction.
///
func (a *Assembly) assembleSCR(tokens []token) []byte {
	if len(tokens) == 0 {
		return []byte{0x00, 0xFB}
	}

	panic("illegal instruction")
}

/// Assemble a SCL instruction.
///
func (a *Assembly) assembleSCL(tokens []token) []byte {
	if len(tokens) == 0 {
		return []byte{0x00, 0xFC}
	}

	panic("illegal instruction")
}

/// Assemble a SYS instruction
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

/// Assemble a JP instruction
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

/// Assemble a CALL instruction
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

/// Assemble a SE instruction
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

/// Assemble a SNE instruction
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

/// Assemble a SKP instruction
///
func (a *Assembly) assembleSKP(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0xE0|byte(x), 0x9E}
	}

	panic("illegal instruction")
}

/// Assemble a SKNP instruction
///
func (a *Assembly) assembleSKNP(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0xE0|byte(x), 0xA1}
	}

	panic("illegal instruction")
}

/// Assemble a OR instruction
///
func (a *Assembly) assembleOR(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[0].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x01}
	}

	panic("illegal instruction")
}

/// Assemble a AND instruction
///
func (a *Assembly) assembleAND(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[0].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x02}
	}

	panic("illegal instruction")
}

/// Assemble a XOR instruction
///
func (a *Assembly) assembleXOR(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[0].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x03}
	}

	panic("illegal instruction")
}

/// Assemble a SHR instruction
///
func (a *Assembly) assembleSHR(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0x80|byte(x), byte(x << 4) | 0x06}
	}

	panic("illegal instruction")
}

/// Assemble a SHL instruction
///
func (a *Assembly) assembleSHL(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V); ok {
		x := ops[0].val.(int)

		return []byte{0x80|byte(x), byte(x << 4) | 0x0E}
	}

	panic("illegal instruction")
}

/// Assemble a ADD instruction
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

/// Assemble a SUB instruction
///
func (a *Assembly) assembleSUB(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x05}
	}

	panic("illegal instruction")
}

/// Assemble a SUBN instruction
///
func (a *Assembly) assembleSUBN(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return []byte{0x80|byte(x), byte(y << 4) | 0x07}
	}

	panic("illegal instruction")
}

/// Assemble a RND instruction
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

/// Assemble a DRW instruction
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

/// Assemble a LD instruction
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

	if ops, ok := a.assembleOperands(tokens, TOKEN_HF, TOKEN_V); ok {
		x := ops[1].val.(int)

		return []byte{0xF0|byte(x), 0x30}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_B, TOKEN_V); ok {
		x := ops[1].val.(int)

		return []byte{0xF0|byte(x), 0x33}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_ADDRESS, TOKEN_V); ok {
		x := ops[1].val.(int)

		if ops[0].val.(token).typ == TOKEN_I {
			return []byte{0xF0|byte(x), 0x55}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_ADDRESS); ok {
		x := ops[0].val.(int)

		if ops[1].val.(token).typ == TOKEN_I {
			return []byte{0xF0|byte(x), 0x65}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_R, TOKEN_V); ok {
		x := ops[1].val.(int)

		if x < 8 {
			return []byte{0xF0|byte(x), 0x75}
		}
	}

	if ops, ok := a.assembleOperands(tokens, TOKEN_V, TOKEN_R); ok {
		x := ops[0].val.(int)

		if x < 8 {
			return []byte{0xF0|byte(x), 0x85}
		}
	}

	panic("illegal instruction")
}

/// Assemble a BYTE instruction
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

/// Assemble a WORD instruction
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

/// Assemble an ALIGN instruction
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

/// Assemble an RESERVE instruction
///
func (a *Assembly) assembleRESERVE(tokens []token) []byte {
	if ops, ok := a.assembleOperands(tokens, TOKEN_LIT); ok {
		n := ops[0].val.(int)

		if n < 0x1000 - len(a.ROM) {
			return make([]byte, n)
		}
	}

	panic("illegal size")
}
