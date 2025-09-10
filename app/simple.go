package main

// Rob Pike's Regex Parser
// Cannot handle nested capture groups and alternations.
// We need 'recursive descent parsing' to handle them
func MatchSequential(line []byte, pattern string) bool {
	if pattern[0] == '^' {
		return matchHere(line, pattern[1:])
	}

	for i := range len(line) {
		if matchHere(line[i:], pattern) {
			return true
		}
	}

	return false
}

func matchHere(input []byte, pattern string) bool {
	switch {
	case len(pattern) == 0:
		return true // no pattern left to match

	case len(pattern) >= 2 && pattern[1] == '*':
		return matchStar(input, pattern[0], pattern[2:])

	case len(pattern) >= 2 && pattern[1] == '?':
		return matchQuestion(input, pattern[0], pattern[2:])

	case len(input) == 0:
		// if input is empty and pattern is 'end anchor', then true
		return pattern == "$"

	case len(pattern) >= 2 && pattern[1] == '+':
		return matchPlus(input, pattern[0], pattern[2:])

	case pattern[0] == input[0] || pattern[0] == '.':
		return matchHere(input[1:], pattern[1:])

	case len(pattern) >= 2 && pattern[0] == '\\':
		return matchShorthand(input, pattern)

	case len(pattern) >= 2 && pattern[0] == '[':
		return matchCharacterClass(input, pattern)

	default:
		return false
	}
}

func matchPlus(input []byte, c byte, pattern string) bool {
	// match the first character
	if len(input) == 0 || (input[0] != c && c != '.') {
		return false
	}

	// match more than one characters
	for i := 0; i < len(input) && (input[i] == c || c == '.'); i++ {
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
	for i := 0; i < len(input) && (input[i] == c || c == '.'); i++ {
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
	if (input[0] == c || c == '.') && matchHere(input[1:], pattern) {
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

	// last char of pattern is assumed to be ']'
	// Todo: this will fail for [abc]+
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
