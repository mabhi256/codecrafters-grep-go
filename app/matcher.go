package main

func matchTokensAt(tokens []string, input []byte, pos int) bool {
	j := pos
	for _, token := range tokens {
		var matched bool
		switch token[0] {
		case '\\':
			matched = matchShorthand(token, input, j)

		case '[':
			matched = matchCharClass(token, input, j)

		case '^':
			matched = matchStartAnchor(token, input, j)

		case '$':
			matched = matchEndAnchor(input, j)

		default:
			matched = matchLiteral(token, input, j)
		}

		if !matched {
			return false
		}
		j++
	}

	return true
}

func matchShorthand(token string, input []byte, pos int) bool {
	if pos >= len(input) {
		return false
	}

	switch token {
	case "\\d":
		return isDigit(input[pos])

	case "\\w":
		return isAlphaNumeric(input[pos])

	case "\\s":
		return input[pos] == ' '

	default:
		return false
	}
}

func matchCharClass(token string, input []byte, pos int) bool {
	if pos >= len(input) {
		return false
	}

	isNegated := token[1] == '^'

	chars := []byte(token[1 : len(token)-1])
	if isNegated {
		chars = chars[1:] // skip the ^
	}

	for _, ch := range chars {
		found := ch == input[pos]

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

func matchStartAnchor(token string, input []byte, pos int) bool {
	if pos >= len(input) {
		return false
	}

	return pos == 0 && token[1] == input[pos]
}

func matchEndAnchor(input []byte, pos int) bool {
	return pos == len(input)
}

func matchLiteral(token string, input []byte, pos int) bool {
	if pos >= len(input) {
		return false
	}

	return token == string(input[pos])
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isAlphaNumeric(char byte) bool {
	isSmall := char >= 'a' && char <= 'z'
	isCapitalized := char >= 'A' && char <= 'Z'

	return isSmall || isCapitalized || isDigit(char) || char == '_'
}
