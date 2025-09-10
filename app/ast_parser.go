package main

import (
	"fmt"
	"strings"
)

type MatchResult struct {
	Success  bool // only used in backtracking
	EndPos   int
	Captures []string // captures[0] = entire match, captures[1] = group 1, etc.
}

type Node interface {
	// traditional AST with backtracking
	match(input []byte, pos int) MatchResult

	// Return ALL possible matches from this position like NFA
	matchAll(input []byte, pos int, captures []string) []MatchResult
}

type SequenceNode struct {
	Children []Node
}

type LiteralNode struct {
	Value byte
}

type CharClassNode struct {
	Chars   []byte
	Negated bool
}

type StartAnchorNode struct{}
type EndAnchorNode struct{}

type QuantifierNode struct {
	Child  Node
	Min    int
	Max    int // -1 for infinity
	Greedy bool
}
type QuantifierMatchResult struct {
	EndPos   int
	Captures []string
}

type DotNode struct{}

type CaptureNode struct {
	Child    Node
	GroupIdx int
}

type AlternationNode struct {
	Children []Node
}

type Parser struct {
	pattern     string
	pos         int // current position in pattern
	nextGroupId int // Track next capture group number
}

func NewParser(pattern string) *Parser {
	return &Parser{
		pattern:     pattern,
		pos:         0,
		nextGroupId: 1, // Groups start at 1 (0 is reserved for entire match)
	}
}

// peek returns current character without advancing
func (p *Parser) peek() byte {
	if p.pos >= len(p.pattern) {
		return 0
	}

	return p.pattern[p.pos]
}

// consumes and returns current character
func (p *Parser) advance() byte {
	if p.pos >= len(p.pattern) {
		return 0
	}

	ch := p.pattern[p.pos]
	p.pos++
	return ch
}

// checks if pattern parsing is done
func (p *Parser) isEOF() bool {
	return p.pos >= len(p.pattern)
}

func parse(pattern string) (Node, int, error) {
	if len(pattern) == 0 {
		return nil, 0, fmt.Errorf("empty pattern")
	}

	parser := NewParser(pattern)
	ast, err := parser.parseExpression()
	return ast, parser.nextGroupId, err
}

func (p *Parser) parseExpression() (Node, error) {
	var alternatives []Node

	for {
		var nodes []Node

		// Collect nodes for this alternative until | or )
		for !p.isEOF() && p.peek() != '|' && p.peek() != ')' {
			atom, err := p.parseQuantified()
			if err != nil {
				return nil, err
			}

			nodes = append(nodes, atom)
		}

		if len(nodes) == 0 {
			return nil, fmt.Errorf("empty pattern")
		}

		var alternative Node
		alternative = SequenceNode{Children: nodes}
		if len(nodes) == 1 {
			alternative = nodes[0]
		}

		alternatives = append(alternatives, alternative)

		// Continue adding alternatives only if we see '|'
		if p.peek() != '|' {
			break
		}
		p.advance()
	}

	// Return single alternative or alternation node
	if len(alternatives) == 1 {
		return alternatives[0], nil
	}
	return AlternationNode{Children: alternatives}, nil
}

func (p *Parser) parseQuantified() (Node, error) {
	atom, err := p.parseAtom()
	if err != nil {
		return nil, err
	}

	// Check for quantifiers
	if p.isEOF() || !isQuantifier(p.peek()) {
		return atom, nil
	}

	// Quantifiers are by default greedy
	// Appending a ? makes it lazy/non-greedy *?, +?, ??, {n,m}?
	ch := p.advance()
	greedy := true
	if p.peek() == '?' {
		p.advance() // consume '?'
		greedy = false
	}

	switch ch {
	case '*':
		return QuantifierNode{Child: atom, Min: 0, Max: -1, Greedy: greedy}, nil

	case '+':
		return QuantifierNode{Child: atom, Min: 1, Max: -1, Greedy: greedy}, nil

	case '?':
		return QuantifierNode{Child: atom, Min: 0, Max: 1, Greedy: greedy}, nil

	default:
		return nil, fmt.Errorf("unexpected quantifier: %c", ch)
	}
}

func isQuantifier(ch byte) bool {
	return ch == '*' || ch == '+' || ch == '?'
}

func (p *Parser) parseAtom() (Node, error) {
	ch := p.advance()

	// Handle escape sequences
	switch ch {
	case '\\':
		return p.parseEscape()

	case '[':
		return p.parseCharClass()

	case '(':
		return p.parseGroup()

	case '^':
		return StartAnchorNode{}, nil

	case '$':
		return EndAnchorNode{}, nil

	case '.':
		return DotNode{}, nil

	default:
		return LiteralNode{Value: ch}, nil
	}
}

func (p *Parser) parseEscape() (Node, error) {
	ch := p.advance()

	switch ch {
	case 'd':
		// \d = [0-9]
		return CharClassNode{Chars: []byte("0123456789"), Negated: false}, nil

	case 'w':
		// \w = [a-zA-Z0-9_]
		wordChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
		return CharClassNode{Chars: []byte(wordChars), Negated: false}, nil

	case 's':
		// \s = [ \t\n\r\f\v] (whitespace)
		return CharClassNode{Chars: []byte(" \t\n\r\f\v"), Negated: false}, nil

	default:
		return LiteralNode{Value: ch}, nil
	}
}

func (p *Parser) parseCharClass() (Node, error) {
	negated := false
	if p.peek() == '^' {
		p.advance()
		negated = true
	}

	var chars []byte
	for !p.isEOF() && p.peek() != ']' {
		char := p.advance()
		chars = append(chars, char)
	}

	if p.isEOF() {
		return nil, fmt.Errorf("expecting ']'")
	}
	p.advance() // consume ']'

	return CharClassNode{Chars: chars, Negated: negated}, nil
}

func (p *Parser) parseGroup() (Node, error) {
	// Assign group number and increment
	groupIdx := p.nextGroupId
	p.nextGroupId++

	// Parse the content inside parentheses
	content, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Expect closing parenthesis
	if p.isEOF() || p.peek() != ')' {
		return nil, fmt.Errorf("expected ')' at position %d", p.pos)
	}
	p.advance() // consume ')'

	return CaptureNode{Child: content, GroupIdx: groupIdx}, nil
}

func printAST(node Node) string {
	return prettyPrint(node, "", true)
}

func prettyPrint(node Node, prefix string, isLast bool) string {
	if node == nil {
		return ""
	}

	var result strings.Builder

	connector := "├─ "
	if isLast {
		connector = "└─ "
	}

	switch node := node.(type) {
	case LiteralNode:
		value := node.Value
		result.WriteString(fmt.Sprintf("%s%sLiteral('%c')\n", prefix, connector, value))

	case StartAnchorNode:
		result.WriteString(fmt.Sprintf("%s%sStartAnchor\n", prefix, connector))
	case EndAnchorNode:
		result.WriteString(fmt.Sprintf("%s%sEndAnchor\n", prefix, connector))

	case DotNode:
		result.WriteString(fmt.Sprintf("%s%sWildcard\n", prefix, connector))

	case CharClassNode:
		negated := ""
		if node.Negated {
			negated = "^"
		}
		result.WriteString(fmt.Sprintf("%s%sCharClass(%s%s)\n", prefix, connector, negated, node.Chars))

	case QuantifierNode:
		maxStr := fmt.Sprintf("%d", node.Max)
		if node.Max == -1 {
			maxStr = "∞"
		}
		result.WriteString(fmt.Sprintf("%s%sQuantifier({%d,%s})\n", prefix, connector, node.Min, maxStr))

		// Child prefix
		childPrefix := prefix
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}

		// Print the child
		result.WriteString(prettyPrint(node.Child, childPrefix, true))

	case SequenceNode:
		result.WriteString(fmt.Sprintf("%s%sSequence\n", prefix, connector))

		// Child prefix
		childPrefix := prefix
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}

		// Print all children
		children := node.Children
		for i, child := range children {
			isLastChild := i == len(children)-1
			result.WriteString(prettyPrint(child, childPrefix, isLastChild))
		}

	case CaptureNode:
		result.WriteString(fmt.Sprintf("%s%sCapture(group_%d)\n", prefix, connector, node.GroupIdx))

		childPrefix := prefix
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}

		result.WriteString(prettyPrint(node.Child, childPrefix, true))
	case AlternationNode:
		result.WriteString(fmt.Sprintf("%s%sAlternation\n", prefix, connector))

		childPrefix := prefix
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}

		// Print all children
		for i, child := range node.Children {
			isLastChild := i == len(node.Children)-1
			result.WriteString(prettyPrint(child, childPrefix, isLastChild))
		}
	}

	return result.String()
}
