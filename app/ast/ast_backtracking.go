package main

import (
	"fmt"
	"strings"
)

func (n SequenceNode) match(input []byte, pos int) MatchResult {
	return n.matchFromChild(input, pos, 0)
}

func (n SequenceNode) matchFromChild(input []byte, pos int, childIdx int) MatchResult {
	if childIdx >= len(n.Children) {
		return MatchResult{
			Success:  true,
			EndPos:   pos,
			Captures: []string{},
		}
	}

	child := n.Children[childIdx]

	// Handle quantifiers specially
	if qNode, ok := child.(QuantifierNode); ok {
		possibleMatches := qNode.getAllMatchesWithCaptures(input, pos)

		// Try each possibility
		for _, qMatch := range possibleMatches {
			restResult := n.matchFromChild(input, qMatch.EndPos, childIdx+1)
			if restResult.Success {
				// Merge captures from quantifier and rest
				merged := mergeCaptures(qMatch.Captures, restResult.Captures)
				return MatchResult{
					Success:  true,
					EndPos:   restResult.EndPos,
					Captures: merged,
				}
			}
		}

		return MatchResult{Success: false, EndPos: pos, Captures: []string{}}
	} else {
		// Regular node - match it first
		matchResult := child.match(input, pos)
		if !matchResult.Success {
			return MatchResult{
				Success:  false,
				EndPos:   pos,
				Captures: []string{},
			}
		}

		// try to match the rest Only if this child matched successfully,
		restResult := n.matchFromChild(input, matchResult.EndPos, childIdx+1)
		if !restResult.Success {
			return restResult
		}

		// Merge captures from both parts
		merged := mergeCaptures(matchResult.Captures, restResult.Captures)
		return MatchResult{
			Success:  true,
			EndPos:   restResult.EndPos,
			Captures: merged,
		}
	}
}

func (n QuantifierNode) getAllMatchesWithCaptures(input []byte, pos int) []QuantifierMatchResult {
	var results []QuantifierMatchResult
	curr := pos
	matchCount := 0
	var currentCaptures []string

	// Try to meet minimum requirement
	for matchCount < n.Min {
		matchResult := n.Child.match(input, curr)
		if !matchResult.Success {
			return nil // Can't meet minimum requirement
		}
		curr = matchResult.EndPos
		matchCount++
		currentCaptures = mergeCaptures(currentCaptures, matchResult.Captures)
	}

	// Add minimum match
	results = append(results, QuantifierMatchResult{
		EndPos:   curr,
		Captures: currentCaptures,
	})

	// Try to match more (up to Max)
	for n.Max == -1 || matchCount < n.Max {
		matchResult := n.Child.match(input, curr)
		if !matchResult.Success {
			break
		}

		curr = matchResult.EndPos
		matchCount++
		currentCaptures = mergeCaptures(currentCaptures, matchResult.Captures)

		results = append(results, QuantifierMatchResult{
			EndPos:   curr,
			Captures: currentCaptures,
		})
	}

	// Order results based on greediness
	if !n.Greedy {
		// Non-greedy: try shorter matches first (already in correct order)
		return results
	} else {
		// Greedy: try longer matches first
		reversed := make([]QuantifierMatchResult, len(results))
		for i, v := range results {
			reversed[len(results)-1-i] = v
		}
		return reversed
	}
}

func mergeCaptures(captures1, captures2 []string) []string {
	if len(captures1) == 0 {
		return captures2
	}
	if len(captures2) == 0 {
		return captures1
	}
	maxLen := max(len(captures1), len(captures2))
	result := make([]string, maxLen)

	// Copy from first captures
	copy(result, captures1)

	// Overwrite with second captures
	copy(result, captures2)

	for i := len(captures2); i < len(captures1); i++ {
		result[i] = ""
	}
	return result
}

func (n LiteralNode) match(input []byte, pos int) MatchResult {
	if pos < len(input) && n.Value == input[pos] {
		return MatchResult{
			Success:  true,
			EndPos:   pos + 1,
			Captures: []string{},
		}
	} else {
		return MatchResult{
			Success:  false,
			EndPos:   pos,
			Captures: []string{},
		}
	}
}

func (n CharClassNode) match(input []byte, pos int) MatchResult {
	if pos >= len(input) {
		return MatchResult{
			Success:  false,
			EndPos:   pos,
			Captures: []string{},
		}
	}

	for _, ch := range n.Chars {
		found := input[pos] == ch
		if found {
			// if negated and we found a match, then return false
			// if not negated and we found a match, then return true
			if n.Negated {
				return MatchResult{
					Success:  false,
					EndPos:   pos,
					Captures: []string{},
				}
			} else {
				return MatchResult{
					Success:  true,
					EndPos:   pos + 1,
					Captures: []string{},
				}
			}
		}
	}

	// if negated and we didn't find anything, then return true
	// if not negated and we didn't find anything, then return false
	if n.Negated {
		return MatchResult{
			Success:  true,
			EndPos:   pos + 1,
			Captures: []string{},
		}
	} else {
		return MatchResult{
			Success:  false,
			EndPos:   pos,
			Captures: []string{},
		}
	}
}

func (n StartAnchorNode) match(input []byte, pos int) MatchResult {
	if pos == 0 {
		// Don't consume input
		return MatchResult{
			Success:  true,
			EndPos:   pos,
			Captures: []string{},
		}
	} else {
		return MatchResult{
			Success:  false,
			EndPos:   pos,
			Captures: []string{},
		}
	}
}

func (n EndAnchorNode) match(input []byte, pos int) MatchResult {
	if pos == len(input) {
		// Don't consume input
		return MatchResult{
			Success:  true,
			EndPos:   pos,
			Captures: []string{},
		}
	} else {
		return MatchResult{
			Success:  false,
			EndPos:   pos,
			Captures: []string{},
		}
	}
}

func (n QuantifierNode) match(input []byte, pos int) MatchResult {
	panic("Should not be called")
}

func (n DotNode) match(input []byte, pos int) MatchResult {
	if pos >= len(input) {
		return MatchResult{
			Success:  false,
			EndPos:   pos,
			Captures: []string{},
		}
	}

	return MatchResult{
		Success:  true,
		EndPos:   pos + 1,
		Captures: []string{},
	}
}

func (n CaptureNode) match(input []byte, pos int) MatchResult {
	result := n.Child.match(input, pos)

	if !result.Success {
		return result
	}

	// Capture what was matched
	capturedText := string(input[pos:result.EndPos])

	// Create captures array big enough for this group
	requiredSize := max(n.GroupIdx+1, len(result.Captures))
	captures := make([]string, requiredSize)
	copy(captures, result.Captures)
	captures[n.GroupIdx] = capturedText

	return MatchResult{
		Success:  true,
		EndPos:   result.EndPos,
		Captures: captures,
	}
}

func (n AlternationNode) match(input []byte, pos int) MatchResult {
	for _, child := range n.Children {
		result := child.match(input, pos)
		if result.Success {
			return result
		}
	}
	return MatchResult{Success: false, EndPos: pos, Captures: []string{}}
}

/*
AST fails for multiple quantifiers (e.g., "a+a+a+a" matching "aaaaaaa"):
- Each quantifier greedily consumes without knowing about later quantifiers
- First a+ takes all 7 'a's, leaving nothing for remaining a+a+a+
- We can use backtracking to handle adjacent nodes, but we can't easily say how much should each quantifier take?
- With 4 quantifiers, there are even more valid splits: (1,1,1,4), (2,1,1,3), (1,2,2,2), etc.
- AST would need to try ALL combinations = exponential complexity

NFA Solution:
- Represents ALL possible consumption patterns as different paths simultaneously
- No "choosing" or backtracking needed - explores all possibilities in parallel
- Linear time complexity regardless of quantifier complexity
*/

// match, captures, _ := MatchAST([]byte("john@example.com"), `(\w+)@(\w+\.\w+)`)
// Result: true, ["john@example.com", "john", "example.com"]
//
//	captures[0] = entire match
//	captures[1] = username
//	captures[2] = domain
func MatchAST(input []byte, pattern string) (bool, []string, error) {
	ast, _, err := parse(pattern)
	if err != nil {
		return false, nil, err
	}
	fmt.Printf("Input: %q\n", string(input))
	fmt.Printf("Pattern: %s\n%s", pattern, printAST(ast))

	if strings.HasPrefix(pattern, "^") {
		matched := ast.match(input, 0)
		if matched.Success {
			entireMatch := string(input[0:matched.EndPos])
			captures := matched.Captures

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
		return false, nil, nil
	}

	// Try matching at each position
	for i := range len(input) {
		matched := ast.match(input, i)

		if matched.Success {
			// Set entire match as group 0
			entireMatch := string(input[i:matched.EndPos])

			captures := matched.Captures
			if len(captures) == 0 {
				captures = []string{entireMatch}
			} else {
				// Insert entire match at index 0
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
