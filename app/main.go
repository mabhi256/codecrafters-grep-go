package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"unicode/utf8"
)

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func hasDigit(line []byte) bool {
	return slices.ContainsFunc(line, isDigit)
}

func isAlphaNumeric(char byte) bool {
	isSmall := char >= 'a' && char <= 'z'
	isCapitalized := char >= 'A' && char <= 'Z'

	return isSmall || isCapitalized || isDigit(char) || char == '_'
}

func hasAlphaNumeric(line []byte) bool {
	return slices.ContainsFunc(line, isAlphaNumeric)
}

// Usage: echo <input_text> | your_program.sh -E <pattern>
func main() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	pattern := os.Args[2]

	line, err := io.ReadAll(os.Stdin) // assume we're only dealing with a single line
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	ok, err := matchLine(line, pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if !ok {
		os.Exit(1)
	}

	// default exit code is 0 which means success
}

func matchLine(line []byte, pattern string) (bool, error) {
	var ok bool

	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	switch {
	case pattern == "\\d":
		ok = hasDigit(line)

	case pattern == "\\w":
		ok = hasAlphaNumeric(line)

	case utf8.RuneCountInString(pattern) == 1:
		ok = bytes.ContainsAny(line, pattern)

	default:
		return false, fmt.Errorf("unsupported pattern: %q", pattern)
	}

	return ok, nil
}
