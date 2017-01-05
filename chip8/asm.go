package chip8

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type tokenType uint

const (
	TOKEN_LABEL tokenType = iota
	TOKEN_COLON
	TOKEN_INSTRUCTION
	TOKEN_V
	TOKEN_B
	TOKEN_I
	TOKEN_F
	TOKEN_K
	TOKEN_DT
	TOKEN_ST
	TOKEN_INDIRECT
	TOKEN_ADDRESS
	TOKEN_LIT
	TOKEN_TEXT
	TOKEN_DELIM
	TOKEN_COMMENT
	TOKEN_EOL
)

type token struct {
	typ tokenType
	val interface{}
}

/// Assemble an input CHIP-8 source code file.
///
func Assemble(file string) []byte {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	// create simple line scanner over the file
	reader := bytes.NewReader(bytes.ToUpper(content))
	scanner := bufio.NewScanner(reader)

	// parsed lines for multiple passes
	lines := make([][]token, 0, 100)

	// the compiled rom, starting at offset 0x200 (for labels)
	rom := make([]byte, 0x200, 0x1000)

	// labels mapped to the compiled byte offset
	labels := make(map[string]int)

	// pass 1 - parse everything, find labels and create the offset map
	for i := 0;scanner.Scan();i++ {
		offset := len(rom)

		// parse the tokens on this line
		tokens, err := readTokens(scanner.Bytes())
		if err != nil {
			panic(fmt.Errorf("(%d): %v", i, err))
		}

		// is there a label on this line, if so, map the offset
		if len(tokens) > 0 {
			if len(tokens) >= 2 && tokens[0].typ == TOKEN_LABEL && tokens[1].typ == TOKEN_COLON {
				labels[tokens[0].val.(string)] = offset

				// drop the label from the tokens, they aren't needed
				tokens = tokens[2:]
			}
		}

		// save the tokens on this line for pass 2 and assemble what's there to track offset
		if len(tokens) > 0 {
			lines = append(lines, tokens)

			// assemble as much as possible, but this work will be thrown away...
			rom = assembleInstruction(tokens, rom, nil)
		}
	}

	// reset the rom so it can be written again with proper labels
	rom = rom[:0x200]

	// pass 2 - assemble all the instructions
	for _, tokens := range lines {
		rom = assembleInstruction(tokens, rom, &labels)
	}

	// output result
	fmt.Println("Assembled", len(rom) - 0x200, "bytes")

	// done, return only the program memory
	return rom[0x200:]
}

/// Parse all the tokens in the line.
///
func readTokens(r []byte) ([]token, error) {
	tokens := make([]token, 0, 20)

	// loop until the entire source is consumed
	for r != nil && len(r) > 0 {
		var t token

		// read the next token in the stream
		t, r = readToken(r)

		if t.typ != TOKEN_COMMENT && t.typ != TOKEN_EOL {
			tokens = append(tokens, t)
		}
	}

	return tokens, nil
}

/// Read the next token.
///
func readToken(r []byte) (token, []byte) {
	for r != nil && len(r) > 0 {
		if c := r[0]; c > 32 {
			switch {
			case c == ':':
				return readSimpleToken(r, TOKEN_COLON)
			case c == ',':
				return readSimpleToken(r, TOKEN_DELIM)
			case c == ']':
				return readSimpleToken(r, TOKEN_ADDRESS)
			case c == '[':
				return readIndirect(r[1:])
			case c == ';':
				return readComment(r[1:])
			case c == '"':
				return readTextLit(r[1:], '"')
			case c == '\'':
				return readTextLit(r[1:], '\'')
			case c == '#':
				return readHexLit(r[1:])
			case c == '$':
				return readBinLit(r[1:])
			case c >= '0' && c <= '9':
				return readDecLit(r)
			case c >= 'A' && c <= 'Z':
				return readLabel(r)
			default:
				panic("syntax error")
			}
		} else {
			r = r[1:]
		}
	}

	return token{typ: TOKEN_EOL}, nil
}

/// Read a simple character token.
///
func readSimpleToken(r []byte, typ tokenType) (token, []byte) {
	if len(r) == 1 {
		return token{typ: typ}, nil
	}

	return token{typ: typ}, r[1:]
}

/// Read a comment string.
///
func readComment(r []byte) (token, []byte) {
	return token{typ: TOKEN_COMMENT, val: string(r)}, nil
}

/// Read an indirect addressed token.
///
func readIndirect(r []byte) (token, []byte) {
	label, i := readToken(r)
	address, a := readToken(i)

	// make sure to close the indirection
	if address.typ != TOKEN_ADDRESS {
		panic("syntax error")
	}

	// wrap the token with an indirection
	return token{typ: TOKEN_INDIRECT, val: label}, a
}

/// Read a label, reference, or register.
///
func readLabel(r []byte) (token, []byte) {
	var i int

	for i = 0;i < len(r);i++ {
		if (r[i] < 'A' || r[i] > 'Z') && (r[i] < '0' || r[i] > '9') && r[i] != '_' {
			break
		}
	}

	// extract the label
	s := string(r[:i])

	// determine whether the label is a register or not
	switch s {
	case "V0":
		return token{typ: TOKEN_V, val: 0}, r[i:]
	case "V1":
		return token{typ: TOKEN_V, val: 1}, r[i:]
	case "V2":
		return token{typ: TOKEN_V, val: 2}, r[i:]
	case "V3":
		return token{typ: TOKEN_V, val: 3}, r[i:]
	case "V4":
		return token{typ: TOKEN_V, val: 4}, r[i:]
	case "V5":
		return token{typ: TOKEN_V, val: 5}, r[i:]
	case "V6":
		return token{typ: TOKEN_V, val: 6}, r[i:]
	case "V7":
		return token{typ: TOKEN_V, val: 7}, r[i:]
	case "V8":
		return token{typ: TOKEN_V, val: 8}, r[i:]
	case "V9":
		return token{typ: TOKEN_V, val: 9}, r[i:]
	case "VA":
		return token{typ: TOKEN_V, val: 10}, r[i:]
	case "VB":
		return token{typ: TOKEN_V, val: 11}, r[i:]
	case "VC":
		return token{typ: TOKEN_V, val: 12}, r[i:]
	case "VD":
		return token{typ: TOKEN_V, val: 13}, r[i:]
	case "VE":
		return token{typ: TOKEN_V, val: 14}, r[i:]
	case "VF":
		return token{typ: TOKEN_V, val: 15}, r[i:]
	case "I":
		return token{typ: TOKEN_I}, r[i:]
	case "B":
		return token{typ: TOKEN_B}, r[i:]
	case "F":
		return token{typ: TOKEN_F}, r[i:]
	case "K":
		return token{typ: TOKEN_K}, r[i:]
	case "D", "DT":
		return token{typ: TOKEN_DT}, r[i:]
	case "S", "ST":
		return token{typ: TOKEN_ST}, r[i:]
	case "CLS", "RET", "LOW", "HIGH", "SYS", "JP", "CALL", "SE", "SNE", "SKP", "SKNP", "LD", "OR", "AND", "XOR", "ADD", "SUB", "SUBN", "SHR", "SHL", "RND", "DRW", "DB", "DW":
		return token{typ: TOKEN_INSTRUCTION, val: s}, r[i:]
	}

	// just a label/reference
	return token{typ: TOKEN_LABEL, val: s}, r[i:]
}

/// Read a decimal literal.
///
func readDecLit(r []byte) (token, []byte) {
	var i int

	for i = 0;i < len(r);i++ {
		if r[i] < '0' || r[i] > '9' {
			break
		}
	}

	// convert the hex value to an unsigned number
	n, _ := strconv.ParseInt(string(r[:i]), 10, 32)

	return token{typ: TOKEN_LIT, val: int(n)}, r[i:]
}

/// Read a hexadecimal literal.
///
func readHexLit(r []byte) (token, []byte) {
	var i int

	for i = 0;i < len(r);i++ {
		if (r[i] < '0' || r[i] > '9') && (r[i] < 'A' || r[i] > 'F') {
			break
		}
	}

	// convert the hex value to an unsigned number
	n, _ := strconv.ParseInt(string(r[0:i]), 16, 32)

	return token{typ: TOKEN_LIT, val: int(n)}, r[i:]
}

/// Read a binary literal.
///
func readBinLit(r []byte) (token, []byte) {
	var i int

	for i = 0;i < len(r);i++ {
		if r[i] != '.' && r[i] != '0' && r[i] > '1' {
			break
		}
	}

	// allow '.' to be considered a '0' in binary
	s := strings.Replace(string(r[0:i]), ".", "0", -1)

	// convert the hex value to an unsigned number
	n, _ := strconv.ParseInt(s, 2, 32)

	return token{typ: TOKEN_LIT, val: int(n)}, r[i:]
}

/// Read a text string literal.
///
func readTextLit(r []byte, term byte) (token, []byte) {
	var i int

	// find the closing quotation
	for i = 0;i < len(r);i++ {
		if r[i] == term {
			break
		}
	}

	// only up to the terminator
	return token{typ: TOKEN_TEXT, val: string(r[1:i-1])}, r[i+1:]
}

/// Assemble a single instruction or data.
///
func assembleInstruction(tokens []token, rom []byte, labels *map[string]int) []byte {
	if tokens[0].typ == TOKEN_INSTRUCTION {
		switch tokens[0].val.(string) {
		case "CLS":
			return assembleCLS(tokens[1:], rom, labels)
		case "RET":
			return assembleRET(tokens[1:], rom, labels)
		case "LOW":
			return assembleLOW(tokens[1:], rom, labels)
		case "HIGH":
			return assembleHIGH(tokens[1:], rom, labels)
		case "SYS":
			return assembleSYS(tokens[1:], rom, labels)
		case "JP":
			return assembleJP(tokens[1:], rom, labels)
		case "CALL":
			return assembleCALL(tokens[1:], rom, labels)
		case "SE":
			return assembleSE(tokens[1:], rom, labels)
		case "SNE":
			return assembleSNE(tokens[1:], rom, labels)
		case "SKP":
			return assembleSKP(tokens[1:], rom, labels)
		case "SKNP":
			return assembleSKNP(tokens[1:], rom, labels)
		case "OR":
			return assembleOR(tokens[1:], rom, labels)
		case "AND":
			return assembleAND(tokens[1:], rom, labels)
		case "XOR":
			return assembleXOR(tokens[1:], rom, labels)
		case "SHR":
			return assembleSHR(tokens[1:], rom, labels)
		case "SHL":
			return assembleSHL(tokens[1:], rom, labels)
		case "ADD":
			return assembleADD(tokens[1:], rom, labels)
		case "SUB":
			return assembleSUB(tokens[1:], rom, labels)
		case "SUBN":
			return assembleSUBN(tokens[1:], rom, labels)
		case "RND":
			return assembleRND(tokens[1:], rom, labels)
		case "DRW":
			return assembleDRW(tokens[1:], rom, labels)
		case "LD":
			return assembleLD(tokens[1:], rom, labels)
		case "DB":
			return assembleDB(tokens[1:], rom, labels)
		case "DW":
			return assembleDW(tokens[1:], rom, labels)
		}
	}

	// syntax error unexpected token
	panic("syntax error")
}

/// Match operand tokens.
///
func matchOperands(tokens []token, labels *map[string]int, m ...tokenType) ([]token, bool) {
	caps := make([]token, 0, 3)

	// loop over all the desired tokens
	for i, t := range m {
		if (i == 0 && len(tokens) == 0) || (i > 0 && len(tokens) < 2) {
			return caps, false
		}

		// if not the first argument, parse a delimiter
		if i > 0 {
			if tokens[0].typ != TOKEN_DELIM {
				return caps, false
			}

			// skip it
			tokens = tokens[1:]
		}

		// match the token type, labels are a special case
		if t == TOKEN_LIT && tokens[0].typ == TOKEN_LABEL {
			if labels != nil {
				if a, ok := (*labels)[tokens[0].val.(string)]; ok {
					caps = append(caps, token{typ: TOKEN_LIT, val: a})
				} else {
					panic("unknown label")
				}
			} else {
				caps = append(caps, token{typ: TOKEN_LIT, val: 0})
			}
		} else if tokens[0].typ != t {
			return caps, false
		} else {
			caps = append(caps, tokens[0])
		}

		// advance to the next token
		tokens = tokens[1:]
	}

	// there should be no tokens left at the end of the match
	return caps, len(tokens) == 0
}

/// Assemble a CLS instruction.
///
func assembleCLS(tokens []token, rom []byte, labels *map[string]int) []byte {
	if len(tokens) > 0 {
		panic("illegal instruction")
	}

	return append(rom, 0x00, 0xE0)
}

/// Assemble a RET instruction.
///
func assembleRET(tokens []token, rom []byte, labels *map[string]int) []byte {
	if len(tokens) > 0 {
		panic("illegal instruction")
	}

	return append(rom, 0x00, 0xEE)
}

/// Assemble a LOW instruction.
///
func assembleLOW(tokens []token, rom []byte, labels *map[string]int) []byte {
	if len(tokens) > 0 {
		panic("illegal instruction")
	}

	return append(rom, 0x00, 0xFE)
}

/// Assemble a HIGH instruction.
///
func assembleHIGH(tokens []token, rom []byte, labels *map[string]int) []byte {
	if len(tokens) > 0 {
		panic("illegal instruction")
	}

	return append(rom, 0x00, 0xFF)
}

/// Assemble a SYS instruction
///
func assembleSYS(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_LIT); ok {
		a := ops[0].val.(int)

		if a < 0x1000 {
			return append(rom, byte(a >> 8 & 0xF), byte(a & 0xFF))
		}
	}

	panic("illegal instruction")
}

/// Assemble a JP instruction
///
func assembleJP(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_LIT); ok {
		a := ops[0].val.(int)

		if a < 0x1000 {
			return append(rom, 0x10|byte(a >> 8 & 0xF), byte(a & 0xFF))
		}
	}

	// might be a jp v0
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_LIT); ok {
		v := ops[0].val.(int)
		a := ops[1].val.(int)

		if v == 0 && a < 0x1000 {
			return append(rom, 0xB0|byte(a >> 8 & 0xF), byte(a & 0xFF))
		}
	}

	panic("illegal instruction")
}

/// Assemble a CALL instruction
///
func assembleCALL(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_LIT); ok {
		a := ops[0].val.(int)

		if a < 0x1000 {
			return append(rom, 0x20|byte(a >> 8 & 0xF), byte(a & 0xFF))
		}
	}

	panic("illegal instruction")
}

/// Assemble a SE instruction
///
func assembleSE(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return append(rom, 0x30|byte(x), byte(b))
		}
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return append(rom, 0x50|byte(x), byte(y << 4))
	}

	panic("illegal instruction")
}

/// Assemble a SNE instruction
///
func assembleSNE(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return append(rom, 0x40|byte(x), byte(b))
		}
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return append(rom, 0x90|byte(x), byte(y << 4))
	}

	panic("illegal instruction")
}

/// Assemble a SKP instruction
///
func assembleSKP(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V); ok {
		x := ops[0].val.(int)

		return append(rom, 0xE0|byte(x), 0x9E)
	}

	panic("illegal instruction")
}

/// Assemble a SKNP instruction
///
func assembleSKNP(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V); ok {
		x := ops[0].val.(int)

		return append(rom, 0xE0|byte(x), 0xA1)
	}

	panic("illegal instruction")
}

/// Assemble a OR instruction
///
func assembleOR(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[0].val.(int)

		return append(rom, 0x80|byte(x), byte(y << 4) | 0x01)
	}

	panic("illegal instruction")
}

/// Assemble a AND instruction
///
func assembleAND(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[0].val.(int)

		return append(rom, 0x80|byte(x), byte(y << 4) | 0x02)
	}

	panic("illegal instruction")
}

/// Assemble a XOR instruction
///
func assembleXOR(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[0].val.(int)

		return append(rom, 0x80|byte(x), byte(y << 4) | 0x03)
	}

	panic("illegal instruction")
}

/// Assemble a SHR instruction
///
func assembleSHR(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V); ok {
		x := ops[0].val.(int)

		return append(rom, 0x80|byte(x), 0x06)
	}

	panic("illegal instruction")
}

/// Assemble a SHL instruction
///
func assembleSHL(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V); ok {
		x := ops[0].val.(int)

		return append(rom, 0x80|byte(x), 0x0E)
	}

	panic("illegal instruction")
}

/// Assemble a ADD instruction
///
func assembleADD(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return append(rom, 0x70|byte(x), byte(b))
		}
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return append(rom, 0x80|byte(x), byte(y << 4) | 0x04)
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_I, TOKEN_V); ok {
		x := ops[0].val.(int)

		return append(rom, 0xF0|byte(x), 0x1E)
	}

	panic("illegal instruction")
}

/// Assemble a SUB instruction
///
func assembleSUB(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return append(rom, 0x80|byte(x), byte(y << 4) | 0x05)
	}

	panic("illegal instruction")
}

/// Assemble a SUBN instruction
///
func assembleSUBN(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return append(rom, 0x80|byte(x), byte(y << 4) | 0x07)
	}

	panic("illegal instruction")
}

/// Assemble a RND instruction
///
func assembleRND(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return append(rom, 0xC0|byte(x), byte(b))
		}
	}

	panic("illegal instruction")
}

/// Assemble a DRW instruction
///
func assembleDRW(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)
		n := ops[2].val.(int)

		if n < 0x10 {
			return append(rom, 0xD0|byte(x), byte(y << 4) | byte(n))
		}
	}

	panic("illegal instruction")
}

/// Assemble a LD instruction
///
func assembleLD(tokens []token, rom []byte, labels *map[string]int) []byte {
	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_LIT); ok {
		x := ops[0].val.(int)
		b := ops[1].val.(int)

		if b < 0x100 {
			return append(rom, 0x60|byte(x), byte(b))
		}
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_V); ok {
		x := ops[0].val.(int)
		y := ops[1].val.(int)

		return append(rom, 0x80|byte(x), byte(y << 4))
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_I, TOKEN_LIT); ok {
		a := ops[1].val.(int)

		if a < 0x1000 {
			return append(rom, 0xA0|byte(a >> 8 & 0xF), byte(a & 0xFF))
		}
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_DT); ok {
		x := ops[0].val.(int)

		return append(rom, 0xF0|byte(x), 0x07)
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_K); ok {
		x := ops[0].val.(int)

		return append(rom, 0xF0|byte(x), 0x0A)
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_DT, TOKEN_V); ok {
		x := ops[1].val.(int)

		return append(rom, 0xF0|byte(x), 0x15)
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_ST, TOKEN_V); ok {
		x := ops[1].val.(int)

		return append(rom, 0xF0|byte(x), 0x18)
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_F, TOKEN_V); ok {
		x := ops[1].val.(int)

		return append(rom, 0xF0|byte(x), 0x29)
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_B, TOKEN_V); ok {
		x := ops[1].val.(int)

		return append(rom, 0xF0|byte(x), 0x33)
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_INDIRECT, TOKEN_V); ok {
		x := ops[1].val.(int)

		if ops[0].val.(token).typ == TOKEN_I {
			return append(rom, 0xF0|byte(x), 0x55)
		}
	}

	if ops, ok := matchOperands(tokens, labels, TOKEN_V, TOKEN_INDIRECT); ok {
		x := ops[0].val.(int)

		if ops[1].val.(token).typ == TOKEN_I {
			return append(rom, 0xF0|byte(x), 0x65)
		}
	}

	panic("illegal instruction")
}

/// Assemble a DB instruction
///
func assembleDB(tokens []token, rom []byte, labels *map[string]int) []byte {
	if len(tokens) == 0 {
		return rom
	}

	// bytes to be written to the rom
	bs := make([]byte, 0, 20)

readByte:
	t := tokens[0]

	// write bytes and strings
	if t.typ  == TOKEN_LIT {
		bs = append(bs, byte(t.val.(int) & 0xFF))
	} else if t.typ == TOKEN_TEXT {
		bs = append(bs, t.val.(string)...)
	} else {
		panic("illegal byte literal")
	}

	// pop this token
	tokens = tokens[1:]

	// is there a delimiter and so we're expecting another token?
	if len(tokens) > 1 && tokens[0].typ == TOKEN_DELIM {
		tokens = tokens[1:]
		goto readByte
	}

	// append all the bytes to the rom
	return append(rom, bs...)
}

/// Assemble a DW instruction
///
func assembleDW(tokens []token, rom []byte, labels *map[string]int) []byte {
	if len(tokens) == 0 {
		return rom
	}

	// bytes to be written to the rom
	bs := make([]byte, 0, 20)

readWord:
	t := tokens[0]

	// write bytes and strings
	if t.typ  == TOKEN_LIT {
		b0 := t.val.(int) >> 8 & 0xFF
		b1 := t.val.(int) & 0xFF

		bs = append(bs, byte(b0), byte(b1))
	} else {
		panic("illegal word literal")
	}

	// pop this token
	tokens = tokens[1:]

	// is there a delimiter and so we're expecting another token?
	if len(tokens) > 1 && tokens[0].typ == TOKEN_DELIM {
		tokens = tokens[1:]
		goto readWord
	}

	// append all the bytes to the rom
	return append(rom, bs...)
}

