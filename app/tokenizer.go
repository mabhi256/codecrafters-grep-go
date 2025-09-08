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
			token, pos := readStartAnchor(pattern, i)
			tokens = append(tokens, token)
			i = pos

		case '$':
			tokens = append(tokens, "$")
			i++

		default:
			tokens = append(tokens, string(pattern[i]))
			i++
		}
	}

	return tokens
}

func readCharClass(pattern string, pos int) (string, int) {
	charClass := ""
	for i := pos; i < len(pattern); i++ {
		charClass += string(pattern[i])
		if pattern[i] == ']' {
			return charClass, i + 1
		}
	}

	return "", len(pattern)
}

func readEscape(pattern string, pos int) (string, int) {
	if len(pattern) == pos+1 {
		return "", len(pattern)
	}

	return pattern[pos : pos+2], pos + 2
}

func readStartAnchor(pattern string, pos int) (string, int) {
	if len(pattern) == pos+1 {
		return "", len(pattern)
	}

	return pattern[pos : pos+2], pos + 2
}
