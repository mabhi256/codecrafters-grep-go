package main

import (
	"fmt"
	"io"
	"os"
)

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
	// tokens := tokenize(pattern)

	// for i := range len(line) {
	// 	matched, err := matchTokensAt(tokens, line, i)
	// 	if err != nil {
	// 		return false, err
	// 	}
	// 	if matched {
	// 		return true, nil
	// 	}
	// }

	if pattern[0] == '^' {
		return matchHere(line, pattern[1:]), nil
	}

	for i := range len(line) {
		if matchHere(line[i:], pattern) {
			return true, nil
		}
	}

	return false, nil
}

func matchHere(input []byte, pattern string) bool {
	if len(pattern) == 0 {
		return true // no pattern left to match
	}

	if len(pattern) >= 2 && pattern[1] == '*' {
		return matchStar(input, pattern[0], pattern[2:])
	}

	if len(pattern) >= 2 && pattern[1] == '?' {
		return matchQuestion(input, pattern[0], pattern[2:])
	}

	if len(input) == 0 {
		// if input is empty and pattern is 'end anchor', then true
		return pattern == "$"
	}

	if len(pattern) >= 2 && pattern[1] == '+' {
		return matchPlus(input, pattern[0], pattern[2:])
	}

	if input[0] == pattern[0] {
		return matchHere(input[1:], pattern[1:])
	}

	if len(pattern) >= 2 && pattern[0] == '\\' {
		return matchShorthand(input, pattern)
	}

	if len(pattern) >= 2 && pattern[0] == '[' {
		return matchCharacterClass(input, pattern)
	}

	return false
}

// Actual gnu grep uses
// - Simple literal strings -> Boyer-Moore
// - Basic regex -> Thompson NFA
// - Complex regex -> Optimized NFA with DFA conversion
// There is a heuristic for determining nfa/dfa

// Rob Pike's Regular Expression Matcher uses implicit AST + backtracking
// But this is only for learning purpose - it cannot handle complex regex
// Something like grep "a*a*a*a*a*a*a*b" huge_file.txt will blowup exponentially

func matchPlus(input []byte, c byte, pattern string) bool {
	// match the first character
	if len(input) == 0 || input[0] != c {
		return false
	}

	// match more than one characters
	for i := 0; i < len(input) && input[i] == c; i++ {
		if matchHere(input[i+1:], pattern) {
			return true
		}
	}

	return false
}

func matchStar(input []byte, c byte, pattern string) bool {
	// match 0 times
	if matchHere(input, pattern) {
		return true
	}

	// If the first n characters match, then match the rest of the pattern
	for i := 0; i < len(input) && input[i] == c; i++ {
		if matchHere(input[i+1:], pattern) {
			return true
		}
	}

	return false
}

func matchQuestion(input []byte, c byte, pattern string) bool {
	// match 0 times
	if matchHere(input, pattern) {
		return true
	}

	// match the first char, then match the rest of the pattern
	if input[0] == c && matchHere(input[1:], pattern) {
		return true
	}

	return false
}

func matchShorthand(input []byte, pattern string) bool {
	switch pattern[1] {
	case 'd':
		return isDigit(input[0]) && matchHere(input[1:], pattern[2:])

	case 'w':
		return isAlphaNumeric(input[0]) && matchHere(input[1:], pattern[2:])

	case 's':
		return input[0] == ' ' && matchHere(input[1:], pattern[2:])

	default:
		return false
	}
}

func matchCharacterClass(input []byte, pattern string) bool {
	isNegated := pattern[1] == '^'

	chars := []byte(pattern[1 : len(pattern)-1])
	if isNegated {
		chars = chars[1:] // skip the ^
	}

	for _, ch := range chars {
		found := ch == input[0]

		if found {
			// if negated and we found a match, then return false
			// if not negated and we found a match, then return true
			return !isNegated
		}
	}

	// if negated and we didn't find anything, then return true
	// if not negated and we didn't find anything, then return false
	return isNegated
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isAlphaNumeric(char byte) bool {
	isSmall := char >= 'a' && char <= 'z'
	isCapitalized := char >= 'A' && char <= 'Z'

	return isSmall || isCapitalized || isDigit(char) || char == '_'
}
