package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/codecrafters-io/grep-starter-go/app/nfa"
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
	if len(os.Args) < 3 || (os.Args[1] != "-E" && os.Args[1] != "-r") {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	pattern := os.Args[2]
	found := false

	if len(os.Args) == 3 {
		found = matchStdin(pattern)
	} else if os.Args[1] == "-r" {
		pattern := os.Args[3]
		dir := os.Args[4]
		found = matchDir(pattern, dir)
	} else {
		for i := 3; i < len(os.Args); i++ {
			fileName := os.Args[i]

			if matchFile(pattern, "", fileName) {
				found = true
			}
		}
	}

	if !found {
		os.Exit(1)
	}
	// default exit code is 0 which means success
}

func matchDir(pattern string, dir string) bool {
	dirEntry, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input dir: %v\n", err)
		os.Exit(2)
	}

	found := false
	for _, entry := range dirEntry {
		foundHere := false
		if entry.IsDir() {
			foundHere = matchDir(pattern, dir+entry.Name()+"/")
		} else {
			foundHere = matchFile(pattern, dir, entry.Name())
		}

		if foundHere {
			found = true
		}
	}

	return found
}

func matchFile(pattern, dir, fileName string) bool {
	if dir != "" {
		fileName = dir + fileName
	}

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input file: %v\n", err)
		os.Exit(2)
	}
	defer file.Close()

	found := false
	scanner := bufio.NewScanner(file)

	// scan line by line
	for scanner.Scan() {
		line := scanner.Text()

		ok, err := matchLine([]byte(line), pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}

		if ok {
			found = true

			if dir != "" || len(os.Args) >= 5 {
				fmt.Printf("%s:%s\n", fileName, line)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
	}

	return found
}

func matchStdin(pattern string) bool {
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

	return ok
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

	// matched, _, err := MatchASTHybrid(line, pattern)
	// if err != nil {
	// 	return false, err
	// }

	matched, err := nfa.MatchNFA(line, pattern)
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
