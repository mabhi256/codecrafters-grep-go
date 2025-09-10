package main

import (
	"fmt"
	"io"
	"os"
)

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isAlphaNumeric(char byte) bool {
	isSmall := char >= 'a' && char <= 'z'
	isCapitalized := char >= 'A' && char <= 'Z'

	return isSmall || isCapitalized || isDigit(char) || char == '_'
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
	// matched := MatchSequential(line, pattern)
	// if matched {
	// 	return true, nil
	// } else {
	// 	return false, nil
	// }

	// matched, _, err := MatchAST(line, pattern)
	// if err != nil {
	// 	return false, err
	// }

	// return matched, nil

	matched, _, err := MatchASTHybrid(line, pattern)
	if err != nil {
		return false, err
	}

	return matched, nil
}

// Actual gnu grep uses
// - Simple literal strings -> Boyer-Moore
// - Basic regex -> Thompson NFA
// - Complex regex -> Optimized NFA with DFA conversion
// There is a heuristic for determining nfa/dfa

// Rob Pike's Regular Expression Matcher uses implicit AST + backtracking
// The recursive call stack in matchHere and matchPlus/Star... is infact AST tree traversal
// But it cannot handle complex regex like nested groups and nested quantifiers.
// Something like grep "a*a*a*a*a*a*a*b" huge_file.txt will blowup exponentially
