package chip8

import (
	"bufio"
	"bytes"
	"io/ioutil"
)

type tokenType uint

const (
	TOKEN_SPACE tokenType = iota
	TOKEN_LABEL
	TOKEN_OP
	TOKEN_V
	TOKEN_B
	TOKEN_I
	TOKEN_F
	TOKEN_K
	TOKEN_DT
	TOKEN_ST
	TOKEN_ADDR
	TOKEN_REF
	TOKEN_DEC
	TOKEN_HEX
	TOKEN_BIN
	TOKEN_TEXT
	TOKEN_DELIM
	TOKEN_COMMENT
)

type token struct {
	typ tokenType
	val interface{}
}

/// Assemble an input CHIP-8 source code file.
///
func Assemble(file string) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	// create simple line scanner over the file
	reader := bytes.NewReader(bytes.ToUpper(content))
	scanner := bufio.NewScanner(reader)

	// loop over each line
	for scanner.Scan() {
		tokens, err := readTokens(scanner.Bytes())
		if err != nil {
			// TODO: panic with err, file & line
		}

		// collect all the tokens read
		for t := range tokens {
			println(t)
		}
	}
}

/// Find the next token in the scanner.
///
func readTokens(r []byte) ([]token, error) {
	tokens := make([]token, 0, 20)

	// loop until the entire source is consumed
	for len(r) > 0 {
		token := nil

		switch c := r[0] {
		case 'A' <= c && c <= 'Z':
			token, r = readLabel(r)
		}

		if token == nil {

		}
	}

	return tokens, nil
}

/// Read a label or reference.
///
func readLabel(r []byte) (token, []byte) {
	for i, c := range r {
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			if c == ':' {
				return token{typ: TOKEN_LABEL, val: r[:i]}, r[i+1:]
			}

			return token{typ: TOKEN_REF, val: r[:i]}, r[i:]
		}
	}

	return nil, nil
}

/// Read a decimal literal.
///
func tokenDecLit(src []byte) (int, []byte, error) {
	for i, c := range src {
		if c < '0' || c > '9' {
			if i > 0 {
				return line[:i], line[i:]
			}

			break
		}
	}

	return nil, line
}

/// Parse a hexadecimal literal.
///
func parseHexLiteral(line []byte) ([]byte, []byte) {
	for i, c := range line[1:] {
		if (c < '0' || c > '9') && (c < 'A' || c > 'F') {
			if i > 0 {
				return line[:i], line[i:]
			}

			break
		}
	}

	return nil, line
}

/// Parse a binary literal.
///
func parseBinLiteral(line []byte) ([]byte, []byte) {
	for i, c := range line[1:] {
		if c != '0' && c != '1' && c != '.' {
			if i > 0 {
				return line[:i], line[i:]
			}

			break
		}
	}

	return nil, line
}


/// Parse whitespace, returns number skipped.
///
func parseSpaces(line []byte) int {
	for i := 0;i < len(line);i++ {
		if line[i] > 32 {
			return i
		}
	}

	return len(line)
}

/// Parse a comment.
///
func parseComment(line []byte) ([]byte, []byte) {
	n := parseSpaces(line)

	// at the end? there's nothing left
	if len(line) == n {
		return nil, []byte{}
	}

	// is there anything left?
	if line[n] == ';' {
		return line[n:], []byte{}
	}

	return nil, line[n:]
}

/// Parse an identifier.
///
func parseIdent(line []byte) ([]byte, []byte) {
	if len(line) == 0 || (line[0] < 'A' || line[0] > 'Z') && line[0] != '_' {
		return nil, line
	}

	// parse all alphanumeric characters
	for i, c := range line {
		if (c < 'A' || c > 'Z') && c != '_' && (c < '0' || c > '9') {
			return line[:i], line[i:]
		}
	}

	// entire line is an identifier
	return line, []byte{}
}

/// Parse any opening label.
///
func parseLabel(line []byte) ([]byte, []byte) {
	label, rest := parseIdent(line)
	if label == nil {
		return nil, line
	}

	// labels must be followed by a colon
	if len(rest) == 0 || rest[0] != ':' {
		return nil, line
	}

	return label, rest[1:]
}

/// Parse an instruction mnemonic.
///
func parseInstruction(line []byte) ([]byte, []byte) {
	if n := parseSpaces(line); n > 0 {
		return parseIdent(line[n:])
	}

	return nil, line
}

/// Parse all operands.
///
func parseOperands(line []byte) ([][]byte, []byte) {
	if n := parseSpaces(line); n > 0 {
		ops := make([][]byte, 0, 3)

		// parse the first operand
		op, r := parseOperand(line[n:])
		if op == nil {
			return nil, r
		}

		for {
			n := parseDelim(r)

			// push the current operand
			ops = append(ops, op)

			// is there another after this one?
			if n == 0 {
				break
			}

			// parse the next operand
			op, r = parseOperand(r[n:])
		}

		return ops, line
	}

	return nil, line
}

/// Parse an operand.
///
func parseOperand(line []byte) ([]byte, []byte) {
	id, r := parseIdent(line)

	if id != nil {
		return id, r
	}

	return parseLiteral(line)
}

/// Parse an operand delimiter (comma).
///
func parseDelim(line []byte) int {
	n := parseSpaces(line)

	// make sure there is a comma
	if len(line) > n && line[n] == ',' {
		return n + 1 + parseSpaces(line[n+1:])
	}

	return 0
}

/// Parse a literal constant.
///
func parseLiteral(line []byte) ([]byte, []byte) {
	if len(line) == 0 {
		return nil, line
	}

	switch line[0] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return parseDecLiteral(line)
	case '#':
		return parseHexLiteral(line)
	case '$':
		return parseBinLiteral(line)
	}

	return parseIdent(line)
}