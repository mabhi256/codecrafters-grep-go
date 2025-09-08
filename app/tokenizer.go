package main

func tokenize(pattern string) []string {
	var tokens []string

	i := 0
	for i < len(pattern) {
		ch := pattern[i]

		switch ch {
		case '[':
			token, pos := readCharClass(pattern, i)
			tokens = append(tokens, token)
			i = pos

		case '\\':
			token, pos := readEscape(pattern, i)
			tokens = append(tokens, token)
			i = pos

		case '^':
			token, pos := readAnchor(pattern, i)
			tokens = append(tokens, token)
			i = pos

		default:
			tokens = append(tokens, string(pattern[i]))
			i++
		}
	}

	return tokens
}

func readCharClass(pattern string, startPos int) (string, int) {
	charClass := ""
	for i := startPos; i < len(pattern); i++ {
		charClass += string(pattern[i])
		if pattern[i] == ']' {
			return charClass, i + 1
		}
	}

	return "", len(pattern)
}

func readEscape(pattern string, startPos int) (string, int) {
	if len(pattern) == startPos+1 {
		return "", len(pattern)
	}

	return pattern[startPos : startPos+2], startPos + 2
}

func readAnchor(pattern string, startPos int) (string, int) {
	if len(pattern) == startPos+1 {
		return "", len(pattern)
	}

	return pattern[startPos : startPos+2], startPos + 2
}
