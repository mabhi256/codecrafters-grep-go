package nfa

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

// Matcher defines the interface for matching input symbols
type Matcher interface {
	Match(input []byte, pos int) bool
	IsEpsilon() bool
}

// LiteralMatcher matches a single literal character
type LiteralMatcher struct {
	Symbol byte
}

func (m LiteralMatcher) Match(input []byte, pos int) bool {
	if pos >= len(input) {
		return false
	}

	return m.Symbol == input[pos]
}

func (m LiteralMatcher) IsEpsilon() bool {
	return false
}

// EpsilonMatcher represents ε-transitions (empty transitions)
type EpsilonMatcher struct{}

func (m EpsilonMatcher) Match(input []byte, pos int) bool {
	return true // ε-transitions always match without consuming input
}

func (m EpsilonMatcher) IsEpsilon() bool {
	return true
}

type CaptureEpsilonMatcher struct {
	CaptureTags []CaptureTag
}

func (m CaptureEpsilonMatcher) Match(input []byte, pos int) bool {
	return true // Always matches (epsilon)
}

func (m CaptureEpsilonMatcher) IsEpsilon() bool {
	return true
}

type CharClassMatcher struct {
	Name    string
	Chars   []byte
	Negated bool
}

func (m CharClassMatcher) Match(input []byte, pos int) bool {
	if pos >= len(input) {
		return false
	}

	found := slices.Contains(m.Chars, input[pos])

	if found != m.Negated { // XOR logic
		return true
	}
	return false
}

func (m CharClassMatcher) IsEpsilon() bool {
	return false
}

type DotMatcher struct{}

func (m DotMatcher) Match(input []byte, pos int) bool {
	if pos >= len(input) {
		return false
	}

	return input[pos] != '\n'
}

func (m DotMatcher) IsEpsilon() bool {
	return false
}

type BackRefMatcher struct {
	GroupID int
}

func (m BackRefMatcher) Match(input []byte, pos int) bool {
	// Cannot access capture data during this stage, because we need to actually run the nfa to get this info
	// Just return true as a placeholder, actual matching will be done in deltaFunction
	return true
}

func (m BackRefMatcher) IsEpsilon() bool {
	return false
}

// CaptureTag represents entering or exiting a capture group
type CaptureTag struct {
	GroupID int
	IsStart bool
}

// Transition represents a labeled edge in the NFA
type Transition struct {
	Target  *State
	Matcher Matcher
}

// State represents a state/node in the NFA graph
type State struct {
	ID          int
	IsAccept    bool
	Transitions []Transition
}

// Global state counter for unique IDs
var stateCounter = -1

func NewState() *State {
	stateCounter++

	return &State{
		ID:          stateCounter,
		IsAccept:    false,
		Transitions: make([]Transition, 0),
	}
}

// AddTransition adds a labeled transition to another state
func (s *State) AddTransition(target *State, matcher Matcher) {
	transition := Transition{
		Target:  target,
		Matcher: matcher,
	}

	s.Transitions = append(s.Transitions, transition)
}

// CaptureGroup represents a captured substring
type CaptureGroup struct {
	Start int
	End   int
	Text  string
}

type MatchResult struct {
	Matched       bool
	CaptureGroups map[int]CaptureGroup // GroupID -> CaptureGroup
}

type ActiveCapture struct {
	GroupID int
	Start   int
}

// RuntimeState represents the current execution state during NFA simulation
type ExecutionContext struct {
	State           *State
	Pos             int // Current position in input
	ActiveCaptures  []ActiveCapture
	CompletedGroups map[int]CaptureGroup
}

// Clone creates a deep copy of the execution context
func (ex *ExecutionContext) Clone() *ExecutionContext {
	clone := &ExecutionContext{
		State:           ex.State,
		Pos:             ex.Pos,
		ActiveCaptures:  make([]ActiveCapture, len(ex.ActiveCaptures)),
		CompletedGroups: make(map[int]CaptureGroup),
	}

	copy(clone.ActiveCaptures, ex.ActiveCaptures)
	maps.Copy(clone.CompletedGroups, ex.CompletedGroups)

	return clone
}

// NFA represents a non-deterministic finite automaton
type NFA struct {
	Start  *State
	Accept *State
}

// NFAParser parses regex patterns directly to NFA using Thompson construction
type NFAParser struct {
	pattern     string
	pos         int
	nextGroupID int // Start at 1 (0 is reserved for full match)
}

// NewNFAParser creates a new parser for the given pattern
func NewNFAParser(pattern string) *NFAParser {
	return &NFAParser{
		pattern:     pattern,
		pos:         0,
		nextGroupID: 1,
	}
}

// peek returns current character without advancing
func (p *NFAParser) peek() byte {
	if p.pos >= len(p.pattern) {
		return 0
	}

	return p.pattern[p.pos]
}

// advance consumes and returns current character
func (p *NFAParser) advance() byte {
	if p.pos >= len(p.pattern) {
		return 0
	}

	ch := p.pattern[p.pos]
	p.pos++
	return ch
}

// isEOF checks if pattern parsing is done
func (p *NFAParser) isEOF() bool {
	return p.pos >= len(p.pattern)
}

// ParseNFA parses the entire pattern and returns an NFA using Thompson construction
func (p *NFAParser) ParseNFA() (*NFA, error) {
	if len(p.pattern) == 0 {
		return nil, fmt.Errorf("empty pattern")
	}

	return p.parseAlternation()
}

func (p *NFAParser) parseAlternation() (*NFA, error) {
	// Parse left atom
	left, err := p.parseSequence()
	if err != nil {
		return nil, err
	}

	// Parse remaining atoms and concatenate
	for !p.isEOF() && p.peek() == '|' {
		p.advance() // consume '|'

		right, err := p.parseSequence()
		if err != nil {
			return nil, err
		}

		left = left.Alternate(right)
	}

	return left, nil
}

// parseSequence handles sequences of atoms
func (p *NFAParser) parseSequence() (*NFA, error) {
	// Parse left atom
	left, err := p.parseQuantifiedAtom()
	if err != nil {
		return nil, err
	}

	// Parse remaining atoms and concatenate
	// Stop if we hit characters '|', ')' that belong to higher-level constructs
	for !p.isEOF() && p.peek() != '|' && p.peek() != ')' {
		right, err := p.parseQuantifiedAtom()
		if err != nil {
			return nil, err
		}

		left = left.Concatenate(right)
	}

	return left, nil
}

func isQuantifier(ch byte) bool {
	return ch == '*' || ch == '+' || ch == '?'
}

func (p *NFAParser) parseQuantifiedAtom() (*NFA, error) {
	// Parse base atom first
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
	// greedy := true
	// if p.peek() == '?' {
	// 	p.advance() // consume '?'
	// 	greedy = false
	// }

	switch ch {
	case '*':
		return p.buildKleeneStar(atom), nil

	case '+':
		return p.buildKleenePlus(atom), nil

	case '?':
		return p.buildOptional(atom), nil

	default:
		return atom, nil
	}
}

//  q₀, q₁, q₂, q₃, q₄

// Alternate combines two NFAs using Thompson construction for alternation
// Thompson rule: N1 | N2 =
//
//	┌──ε──▶ ( N1 ) ─ε──▶┐
//	q₀              	   q₁
//	└──ε──▶ ( N2 ) ─ε──▶┘
func (nfa1 *NFA) Alternate(nfa2 *NFA) *NFA {
	q0 := NewState() // Start state q0
	q1 := NewState() // Accept state q1
	q1.IsAccept = true

	// ε-transitions from q0 (new start) to both alts
	q0.AddTransition(nfa1.Start, EpsilonMatcher{})
	q0.AddTransition(nfa2.Start, EpsilonMatcher{})

	// ε-transitions from both alts to q1 (new accept)
	nfa1.Accept.AddTransition(q1, EpsilonMatcher{})
	nfa2.Accept.AddTransition(q1, EpsilonMatcher{})

	// Old accept states are no longer accepting
	nfa1.Accept.IsAccept = false
	nfa2.Accept.IsAccept = false

	return &NFA{Start: q0, Accept: q1}
}

// Kleene Star: a*
//
//					  ┌─────ε─────┐
//					  ▼			  │
// Pattern: q₀ --ε--> q₁--(atom)->q₂ --ε--> q₃
//		    │								 ▲
//          └────────────────ε───────────────┘
//
// States:
// q0: New start state
// q1: Original atom's start state
// q2: Original atom's accept state (no longer accepting)
// q3: New final accept state

// Transitions:
// q0 --ε--> q1 (enter the 'a' pattern)
// q0 --ε--> q3 (skip 'a' entirely - zero occurrences)
// q1 --'a'--> q2 (original atom transition)
// q2 --ε--> q1 (loop back to match another 'a')
// q2 --ε--> q3 (exit after matching some 'a's)
func (p *NFAParser) buildKleeneStar(nfa *NFA) *NFA {
	q0 := NewState() // Start state q0
	q3 := NewState() // Accept state q3
	q3.IsAccept = true

	// q0 --ε--> q1 (enter the 'a' pattern)
	q0.AddTransition(nfa.Start, EpsilonMatcher{})

	// q0 --ε--> q3 (skip 'a' entirely - 0 occurrences)
	q0.AddTransition(q3, EpsilonMatcher{})

	// q2 --ε--> q1 (loop back to match another 'a')
	nfa.Accept.AddTransition(nfa.Start, EpsilonMatcher{})

	// q2 --ε--> q3 (exit after matching some 'a's)
	nfa.Accept.AddTransition(q3, EpsilonMatcher{})

	nfa.Accept.IsAccept = false
	return &NFA{Start: q0, Accept: q3}
}

// Kleene Plus: a+
//
//					  ┌─────ε─────┐
//					  ▼			  │
// Pattern: q₀ --ε--> q₁--(atom)->q₂ --ε--> q₃
//
// States:
// q0: New start state
// q1: Original atom's start state
// q2: Original atom's accept state (no longer accepting)
// q3: New final accept state

// Transitions:
// q0 --ε--> q1 (enter the 'a' pattern)
// q1 --'a'--> q2 (original atom transition)
// q2 --ε--> q1 (loop back to match another 'a')
// q2 --ε--> q3 (exit after matching some 'a's)
func (p *NFAParser) buildKleenePlus(atom *NFA) *NFA {
	q0 := NewState() // Start state q0
	q3 := NewState() // Accept state q3
	q3.IsAccept = true

	// q0 --ε--> q1 (enter the 'a' pattern)
	q0.AddTransition(atom.Start, EpsilonMatcher{})

	// q2 --ε--> q1 (loop back to match another 'a')
	atom.Accept.AddTransition(atom.Start, EpsilonMatcher{})

	// q2 --ε--> q3 (exit after matching some 'a's)
	atom.Accept.AddTransition(q3, EpsilonMatcher{})

	atom.Accept.IsAccept = false
	return &NFA{Start: q0, Accept: q3}
}

// Optional: a?
//
// Pattern: q₀ --ε--> q₁--(atom)->q₂ --ε--> q₃
//		    │								 ▲
//          └────────────────ε───────────────┘
//
// States:
// q0: New start state
// q1: Original atom's start state
// q2: Original atom's accept state (no longer accepting)
// q3: New final accept state

// Transitions:
// q0 --ε--> q3 (skip 'a' entirely - zero occurrences)
// q0 --ε--> q1 (enter the 'a' pattern)
// q1 --'a'--> q2 (original atom transition)
// q2 --ε--> q3 (exit after matching some 'a's)
func (p *NFAParser) buildOptional(atom *NFA) *NFA {
	q0 := NewState() // Start state q0
	q3 := NewState() // Accept state q3
	q3.IsAccept = true

	// q0 --ε--> q3 (skip 'a' entirely - zero occurrences)
	q0.AddTransition(q3, EpsilonMatcher{})

	// q0 --ε--> q1 (enter the 'a' pattern)
	q0.AddTransition(atom.Start, EpsilonMatcher{})

	// q2 --ε--> q3 (exit after matching some 'a's)
	atom.Accept.AddTransition(q3, EpsilonMatcher{})

	atom.Accept.IsAccept = false
	return &NFA{Start: q0, Accept: q3}
}

// Concatenate combines two NFAs
// Thompson rule: N1 · N2 = add ε-transition from N1.Accept to N2.Start
//
// Before: N1: q₀ --a--> q1(accept)  N2: q2 --b--> q3(accept)
// After:  q₀ --a--> q1 --ε--> q2 --b--> q3(accept)
func (nfa1 *NFA) Concatenate(nfa2 *NFA) *NFA {
	// Add ε-transition from nfa1's accept to nfa2's start
	nfa1.Accept.AddTransition(nfa2.Start, EpsilonMatcher{})

	// nfa1's accept state is no longer accepting
	nfa1.Accept.IsAccept = false

	// Result NFA: start from nfa1, accept at nfa2
	return &NFA{
		Start:  nfa1.Start,
		Accept: nfa2.Accept,
	}
}

// parseAtom creates an NFA fragment for a single atom
// Uses Thompson construction: exactly one initial state and one final state
func (p *NFAParser) parseAtom() (*NFA, error) {
	if p.isEOF() {
		return nil, fmt.Errorf("unexpected end of pattern")
	}

	symbol := p.advance()

	switch symbol {
	case '\\':
		return p.parseEscape()

	case '[':
		return p.parseCharClass()

	case '.':
		return p.buildDotNFA(), nil

	case '(':
		return p.parseGroup()

	default:
		return p.buildLiteralNFA(symbol), nil
	}
}

func (p *NFAParser) parseGroup() (*NFA, error) {
	currGroupID := p.nextGroupID
	p.nextGroupID++ // Ready for next group

	// Parse content inside parentheses
	nfa, err := p.parseAlternation()
	if err != nil {
		return nil, err
	}

	if p.peek() != ')' {
		return nil, fmt.Errorf("expected ')', found: %c", p.peek())
	}
	if p.isEOF() {
		return nil, fmt.Errorf("unexpected EOF")
	}
	p.advance() // consume ')'

	// Now build: q0 --'('-→ NFA --')'-→ q1
	q0 := NewState()
	q1 := NewState()
	q1.IsAccept = nfa.Accept.IsAccept

	startTag := CaptureTag{GroupID: currGroupID, IsStart: true}
	q0.AddTransition(nfa.Start, CaptureEpsilonMatcher{CaptureTags: []CaptureTag{startTag}})

	endTag := CaptureTag{GroupID: currGroupID, IsStart: false}
	nfa.Accept.AddTransition(q1, CaptureEpsilonMatcher{CaptureTags: []CaptureTag{endTag}})
	nfa.Accept.IsAccept = false

	return &NFA{Start: q0, Accept: q1}, nil
}

func (p *NFAParser) buildDotNFA() *NFA {
	q0 := NewState()
	q1 := NewState()
	q1.IsAccept = true

	q0.AddTransition(q1, DotMatcher{})

	return &NFA{Start: q0, Accept: q1}
}

func (p *NFAParser) parseEscape() (*NFA, error) {
	var nfa *NFA
	symbol := p.advance()
	switch {
	case symbol == 'd':
		// \d = [0-9]
		nfa = p.buildCharClassNFA("\\d", []byte("0123456789"), false)

	case symbol == 'w':
		// \w = [a-zA-Z0-9_]
		wordChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
		nfa = p.buildCharClassNFA("\\w", []byte(wordChars), false)

	case symbol == 's':
		// \s = [ \t\n\r\f\v] (whitespace)
		nfa = p.buildCharClassNFA("\\s", []byte(" \t\n\r\f\v"), false)

	case isDigit(symbol):
		groupID, err := p.parseBackreferenceGroupID(symbol)
		if err != nil {
			return nil, err
		}

		nfa = p.buildBackReference(groupID)

	default:
		nfa = p.buildLiteralNFA(symbol)
	}

	return nfa, nil
}

func (p *NFAParser) parseBackreferenceGroupID(digit byte) (int, error) {
	digits := string(digit)
	for !p.isEOF() && isDigit(p.peek()) {
		nextDigit := p.advance()
		digits += string(nextDigit)
	}

	groupID, err := strconv.Atoi(digits)
	if err != nil {
		return 0, err
	}

	if groupID == 0 {
		return 0, fmt.Errorf("invalid backreference \\0")
	}

	return groupID, nil
}

func (p *NFAParser) buildBackReference(groupID int) *NFA {
	q0 := NewState() // Start state
	q1 := NewState() // Accept state
	q1.IsAccept = true

	// Create backreference matcher with integer group ID
	matcher := BackRefMatcher{GroupID: groupID}
	q0.AddTransition(q1, matcher)

	return &NFA{
		Start:  q0,
		Accept: q1,
	}
}

func (p *NFAParser) parseCharClass() (*NFA, error) {
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

	name := fmt.Sprintf("[%s]", chars)

	return p.buildCharClassNFA(name, chars, negated), nil
}

func (p *NFAParser) buildCharClassNFA(name string, chars []byte, negated bool) *NFA {
	q0 := NewState() // Start state
	q1 := NewState() // Accept state
	q1.IsAccept = true

	matcher := CharClassMatcher{Name: name, Chars: chars, Negated: negated}
	q0.AddTransition(q1, matcher)

	return &NFA{
		Start:  q0,
		Accept: q1,
	}
}

// buildLiteralNFA creates Thompson NFA fragment for a literal
// Thompson construction ensures: one start state, one accept state
//
// Structure: q₀ --symbol-→ q₁ (accept)
func (p *NFAParser) buildLiteralNFA(symbol byte) *NFA {
	q0 := NewState() // Start state
	q1 := NewState() // Accept state
	q1.IsAccept = true

	// Add transition: δ(q₀, symbol) = {q₁}
	matcher := LiteralMatcher{Symbol: symbol}
	q0.AddTransition(q1, matcher)

	return &NFA{
		Start:  q0,
		Accept: q1,
	}
}

// deltaFunction implements the NFA transition function δ
// δ: Q × Σ → P(Q) (power set of states)
func deltaFunction(contexts []*ExecutionContext, input []byte) []*ExecutionContext {
	nextContexts := []*ExecutionContext{}

	for _, ctx := range contexts {
		for _, transition := range ctx.State.Transitions {
			// If BackRefMatcher matches we advance pos by len(text)
			if matcher, ok := transition.Matcher.(BackRefMatcher); ok {
				group, exists := ctx.CompletedGroups[matcher.GroupID]
				if !exists {
					continue
				}

				endPos := ctx.Pos + len(group.Text)
				if endPos > len(input) {
					continue
				}

				if group.Text == string(input[ctx.Pos:endPos]) {
					newCtx := ctx.Clone()
					newCtx.State = transition.Target
					newCtx.Pos += len(group.Text)
					nextContexts = append(nextContexts, newCtx)
				}

				continue
			}

			// Check non-epsilon transitions only
			if !transition.Matcher.IsEpsilon() &&
				transition.Matcher.Match(input, ctx.Pos) {

				newCtx := ctx.Clone()
				newCtx.State = transition.Target
				newCtx.Pos++

				if matcher, ok := transition.Matcher.(CaptureEpsilonMatcher); ok {
					newCtx.ApplyTags(matcher.CaptureTags, input)
				}

				nextContexts = append(nextContexts, newCtx)
			}
		}
	}

	return nextContexts
}

// epsilonClosure computes ε-closure of a set of states
// ε-closure(S) = set of states reachable from S using only ε-transitions
func epsilonClosure(contexts []*ExecutionContext, input []byte) []*ExecutionContext {
	closure := make([]*ExecutionContext, 0)
	visited := make(map[*State]bool)
	stack := slices.Clone(contexts)

	// Use DFS to follow all ε-paths
	for len(stack) > 0 {
		// pop from end (because append pushes from end)
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current.State] {
			continue
		}

		visited[current.State] = true
		closure = append(closure, current)

		// Follow all ε-transitions
		for _, transition := range current.State.Transitions {
			if transition.Matcher.IsEpsilon() && !visited[transition.Target] {
				newCtx := current.Clone()
				newCtx.State = transition.Target

				// Apply capture tags if this is a CaptureEpsilonMatcher
				if matcher, ok := transition.Matcher.(CaptureEpsilonMatcher); ok {
					newCtx.ApplyTags(matcher.CaptureTags, input)
				}

				stack = append(stack, newCtx)
			}
		}
	}

	return closure
}

// ApplyTags applies capture tags to the current execution context (runtime state)
func (ex *ExecutionContext) ApplyTags(tags []CaptureTag, input []byte) {
	for _, tag := range tags {
		if tag.IsStart {
			// Start a new capture
			capture := ActiveCapture{
				GroupID: tag.GroupID,
				Start:   ex.Pos,
			}
			ex.ActiveCaptures = append(ex.ActiveCaptures, capture)
		} else {
			// End a capture - find the most recent start for this group
			for i := len(ex.ActiveCaptures) - 1; i >= 0; i-- {
				capture := ex.ActiveCaptures[i]
				if capture.GroupID == tag.GroupID {

					// Add to completed capture
					ex.CompletedGroups[capture.GroupID] = CaptureGroup{
						Start: capture.Start,
						End:   ex.Pos,
						Text:  string(input[capture.Start:ex.Pos]),
					}

					// Remove from active captures
					ex.ActiveCaptures = append(ex.ActiveCaptures[:i], ex.ActiveCaptures[i+1:]...)
					break
				}
			}
		}
	}
}

// RunNFA executes the NFA against the input
func (nfa *NFA) Run(input []byte, pos int, hasEndAnchor bool) *MatchResult {
	currContexts := []*ExecutionContext{
		{
			State:           nfa.Start,
			Pos:             pos,
			ActiveCaptures:  make([]ActiveCapture, 0),
			CompletedGroups: make(map[int]CaptureGroup),
		},
	}

	// Apply ε-closure to initial context
	currContexts = epsilonClosure(currContexts, input)

	for {
		// For each context, try all transitions
		currContexts = deltaFunction(currContexts, input)
		if len(currContexts) == 0 {
			return &MatchResult{Matched: false} // cannot proceed further
		}

		// Apply ε-closure after each transition
		currContexts = epsilonClosure(currContexts, input)

		// Check if any current state is a final state
		for _, ctx := range currContexts {
			if ctx.State.IsAccept {
				ctx.CompletedGroups[0] = CaptureGroup{
					Start: pos,
					End:   ctx.Pos,
					Text:  string(input[pos:ctx.Pos]),
				}

				if hasEndAnchor {
					if ctx.Pos == len(input) {
						return &MatchResult{
							Matched:       true,
							CaptureGroups: ctx.CompletedGroups,
						}
					}
				} else {
					return &MatchResult{
						Matched:       true,
						CaptureGroups: ctx.CompletedGroups,
					}
				}
			}
		}
	}
}

func MatchNFA(input []byte, pattern string) (bool, error) {
	hasStartAnchor := pattern[0] == '^'
	hasEndAnchor := pattern[len(pattern)-1] == '$'

	// Strip anchors from pattern before parsing
	if hasStartAnchor {
		pattern = pattern[1:]
	}
	if hasEndAnchor {
		pattern = pattern[:len(pattern)-1]
	}

	parser := NewNFAParser(pattern)
	nfa, err := parser.ParseNFA()
	if err != nil {
		return false, err
	}

	if hasStartAnchor {
		result := nfa.Run(input, 0, hasEndAnchor)
		if result.Matched {
			return true, nil
		}
	} else {
		for i := range len(input) {
			result := nfa.Run(input, i, hasEndAnchor)
			if result.Matched {
				return true, nil
			}
		}
	}

	return false, nil
}
