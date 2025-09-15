package main

import (
	"fmt"
	"slices"
	"strings"
)

func (n SequenceNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	return n.matchAllChildren(input, pos, captures, 0)
}

func (n SequenceNode) matchAllChildren(input []byte, pos int, captures []string, childIdx int) []MatchResult {
	// Base case: matched all children
	if childIdx >= len(n.Children) {
		return []MatchResult{{EndPos: pos, Captures: slices.Clone(captures)}}
	}

	var allResults []MatchResult
	child := n.Children[childIdx]

	// Get all possible matches for current child
	childMatches := child.matchAll(input, pos, captures)

	// For each child match, try to match remaining children
	for _, childMatch := range childMatches {
		restResults := n.matchAllChildren(input, childMatch.EndPos, childMatch.Captures, childIdx+1)
		allResults = append(allResults, restResults...)
	}

	return allResults
}

func (n LiteralNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	if pos < len(input) && n.Value == input[pos] {
		return []MatchResult{{EndPos: pos + 1, Captures: slices.Clone(captures)}}
	}
	return nil
}

func (n CharClassNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	if pos >= len(input) {
		return nil
	}

	found := slices.Contains(n.Chars, input[pos])

	if found != n.Negated { // XOR logic
		return []MatchResult{{EndPos: pos + 1, Captures: slices.Clone(captures)}}
	}
	return nil
}

func (n StartAnchorNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	if pos == 0 {
		return []MatchResult{{EndPos: pos, Captures: slices.Clone(captures)}}
	}
	return nil
}

func (n EndAnchorNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	if pos == len(input) {
		return []MatchResult{{EndPos: pos, Captures: slices.Clone(captures)}}
	}
	return nil
}

func (n DotNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	if pos >= len(input) {
		return nil
	}
	return []MatchResult{{EndPos: pos + 1, Captures: slices.Clone(captures)}}
}

func (n CaptureNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	childMatches := n.Child.matchAll(input, pos, captures)

	for i, match := range childMatches {
		newCaptures := slices.Clone(match.Captures)
		newCaptures[n.GroupIdx] = string(input[pos:match.EndPos])
		childMatches[i].Captures = newCaptures
	}

	return childMatches
}

func (n AlternationNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	var allResults []MatchResult
	for _, child := range n.Children {
		childResults := child.matchAll(input, pos, captures)
		allResults = append(allResults, childResults...)
	}
	return allResults
}

func (n QuantifierNode) matchAll(input []byte, pos int, captures []string) []MatchResult {
	var allResults []MatchResult

	// Try each possible match count from Min to Max
	for matchCount := n.Min; n.Max == -1 || matchCount <= n.Max; matchCount++ {
		exactResults := n.matchExactly(input, pos, captures, matchCount)
		if len(exactResults) == 0 {
			break // Can't match this many, so we can't match any more
		}
		allResults = append(allResults, exactResults...)
	}

	if len(allResults) == 0 {
		return nil
	}

	// Handle greediness by ordering results
	if n.Greedy {
		slices.Reverse(allResults) // Greedy: longer matches first
	}

	// Non-greedy: shorter matches first (already correct order)
	return allResults
}

// Helper method to match exactly N occurrences
func (n QuantifierNode) matchExactly(input []byte, pos int, captures []string, count int) []MatchResult {
	if count == 0 {
		// For 0 matches, we need to handle capture groups that should be cleared
		if captureNode, ok := n.Child.(CaptureNode); ok {
			// For capture groups that match 0 times, set the capture to ""
			newCaptures := slices.Clone(captures)
			newCaptures[captureNode.GroupIdx] = ""
			return []MatchResult{{EndPos: pos, Captures: newCaptures}}
		}
		// For non-capture nodes, 0 matches means no change
		return []MatchResult{{EndPos: pos, Captures: slices.Clone(captures)}}
	}

	currentResults := []MatchResult{{EndPos: pos, Captures: captures}}

	for range count {
		var nextResults []MatchResult
		for _, result := range currentResults {
			matches := n.Child.matchAll(input, result.EndPos, result.Captures)
			nextResults = append(nextResults, matches...)
		}

		if len(nextResults) == 0 {
			return nil // Can't complete the required matches
		}
		currentResults = nextResults
	}

	return currentResults
}

func MatchASTHybrid(input []byte, pattern string) (bool, []string, error) {
	ast, numGroups, err := parse(pattern)
	if err != nil {
		return false, nil, err
	}

	fmt.Printf("Input: %q\n", string(input))
	fmt.Printf("Pattern: %s\n%s", pattern, printAST(ast))

	initCaptures := make([]string, numGroups)

	// Determine starting positions
	positions := []int{0}
	if !strings.HasPrefix(pattern, "^") {
		positions = make([]int, len(input))
		for i := range positions {
			positions[i] = i
		}
	}

	// Try each position
	for _, pos := range positions {
		results := ast.matchAll(input, pos, initCaptures)
		if len(results) > 0 {
			result := results[0]
			entireMatch := string(input[pos:result.EndPos])

			captures := result.Captures
			if len(captures) == 0 {
				captures = []string{entireMatch}
			} else {
				captures[0] = entireMatch
			}

			fmt.Printf("found %d groups:\n", len(captures))
			for i, group := range captures {
				fmt.Printf("%d: %q\n", i, group)
			}

			return true, captures, nil
		}
	}

	return false, nil, nil
}
