package chip8

import (
	"io/ioutil"
	"regexp"
	"bytes"
	"fmt"
	"unicode"
)

/// Token types.
///
const (
	COMMENT = iota
	LABEL
	VREG
	LITERAL
	TEXT
)

/// Lexical token patterns.
///
var (
	reComment = regexp.MustCompile(`;.*`)
	reLabel = regexp.MustCompile(`\a[\a\d_]*:`)
	reInst = regexp.MustCompile(`\a+`)
	reReg = regexp.MustCompile(`V[\dA-F]|I|K|F|B|DT|ST`)
	reHexLit = regexp.MustCompile(`#[\dA-F]+`)
	reBinLit = regexp.MustCompile(`$[01.]`)
	reDecLit = regexp.MustCompile(`\d+`)
	reText = regexp.MustCompile(`'([^\\']+|\\.)+'`)
)

/// Source tokens.
///
type Token struct {
	kind uint
	s    []string
}

/// Assemble an input CHIP-8 source code file.
///
func Assemble(file string) ([]byte, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// split the bytes into lines
	lines := bytes.Split(content, []byte{'\n'})

	// pass 1 - loop over lines, tokenize
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}

		//
		label, mnemonic, err := Tokenize(line)
		if err != nil {
			return nil, fmt.Errorf("%s (%d): %v", file, i, err)
		}

		fmt.Println("Label:", label, "Op:", mnemonic)
	}

	return nil, nil
}

/// Walk a string and tokenize each aspect of it.
///
func Tokenize(line []byte) (string, string, error) {
	label, r1 := parseLabel(line)
	mnemonic, _ := parseMnemonic(r1)

	return label, mnemonic, nil
}

/// Parse proceeding whitespace.
///
func parseSpaces(line []byte) (bool, []byte) {
	i := bytes.IndexFunc(line, func (r rune) bool {
		return !unicode.IsSpace(r)
	})

	if i < 0 {
		return false, line
	}

	return true, line[i:]
}

/// Parse any opening label.
///
func parseLabel(line []byte) (string, []byte) {
	return "", line
}

/// Parse an instruction mnemonic.
///
func parseMnemonic(line []byte) (string, []byte) {
	spaces, rest := parseSpaces(line)
	if !spaces {
		return "", line
	}

	// find the first non-letter
	i := bytes.IndexFunc(rest, func (r rune) bool {
		return !unicode.IsLetter(r)
	})

	if i < 0 {
		return "", line
	}

	return string(rest[:i]), rest[i:]
}
