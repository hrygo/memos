package memo

import (
	"strings"
	"unicode"
)

// SnippetExtractor provides smart context extraction around matched positions.
// P1-C002: Context-aware snippet extraction with word boundary adjustment.
type SnippetExtractor struct {
	defaultContextChars int
	maxContextChars     int
}

// NewSnippetExtractor creates a new SnippetExtractor with default settings.
func NewSnippetExtractor() *SnippetExtractor {
	return &SnippetExtractor{
		defaultContextChars: 50,
		maxContextChars:     200,
	}
}

// ExtractOptions configures snippet extraction behavior.
type ExtractOptions struct {
	ContextChars int  // Characters to include before/after match center
	AddEllipsis  bool // Whether to add "..." for truncated content
}

// DefaultExtractOptions returns sensible defaults.
func DefaultExtractOptions() *ExtractOptions {
	return &ExtractOptions{
		ContextChars: 50,
		AddEllipsis:  true,
	}
}

// ExtractSnippet extracts a context-aware snippet around the best match position.
// Returns the snippet text and adjusted highlight positions within the snippet.
func (e *SnippetExtractor) ExtractSnippet(
	content string,
	matches []Highlight,
	opts *ExtractOptions,
) (string, []Highlight) {
	if opts == nil {
		opts = DefaultExtractOptions()
	}

	// Apply defaults and constraints
	contextChars := opts.ContextChars
	if contextChars <= 0 {
		contextChars = e.defaultContextChars
	}
	if contextChars > e.maxContextChars {
		contextChars = e.maxContextChars
	}

	contentRunes := []rune(content)
	contentLen := len(contentRunes)

	// No content
	if contentLen == 0 {
		return "", nil
	}

	// No matches: return beginning of content
	if len(matches) == 0 {
		return e.extractFromStart(contentRunes, contextChars*2, opts.AddEllipsis), nil
	}

	// Select best match point (first match = most relevant)
	bestMatch := e.selectBestMatch(matches)

	// Calculate window around match center
	center := bestMatch.Start
	start, end := e.calculateWindow(center, contentLen, contextChars)

	// Adjust to word boundaries
	start = e.adjustToWordBoundary(contentRunes, start, false)
	end = e.adjustToWordBoundary(contentRunes, end, true)

	// Build snippet with optional ellipsis
	snippet, prefixLen := e.buildSnippet(contentRunes, start, end, opts.AddEllipsis)

	// Adjust highlight positions for the snippet
	adjustedMatches := e.adjustMatchPositions(matches, start, end, prefixLen)

	return snippet, adjustedMatches
}

// extractFromStart extracts from the beginning when no matches exist.
func (e *SnippetExtractor) extractFromStart(runes []rune, maxLen int, addEllipsis bool) string {
	runeLen := len(runes)
	end := maxLen
	if end > runeLen {
		end = runeLen
	}

	// Adjust to word boundary
	end = e.adjustToWordBoundary(runes, end, true)

	snippet := string(runes[:end])
	if addEllipsis && end < runeLen {
		snippet += "..."
	}
	return snippet
}

// selectBestMatch chooses the best match point for centering the snippet.
// Currently selects the first match (earliest position = most relevant).
func (e *SnippetExtractor) selectBestMatch(matches []Highlight) Highlight {
	if len(matches) == 0 {
		return Highlight{}
	}

	// Strategy: use first match (closest to beginning = typically most relevant)
	// Future enhancement: could consider match density or score weighting
	return matches[0]
}

// calculateWindow computes the start and end positions for the snippet window.
func (e *SnippetExtractor) calculateWindow(center, contentLen, contextChars int) (start, end int) {
	start = center - contextChars
	end = center + contextChars

	// Clamp to content bounds
	if start < 0 {
		// Shift window right if we hit the start
		end += -start
		start = 0
	}
	if end > contentLen {
		// Shift window left if we hit the end
		start -= end - contentLen
		end = contentLen
	}
	// Re-clamp start after shift
	if start < 0 {
		start = 0
	}

	return start, end
}

// adjustToWordBoundary adjusts position to the nearest word boundary.
// For start positions (isEnd=false), moves backward to find separator.
// For end positions (isEnd=true), moves forward to find separator.
func (e *SnippetExtractor) adjustToWordBoundary(runes []rune, pos int, isEnd bool) int {
	runeLen := len(runes)
	if pos <= 0 {
		return 0
	}
	if pos >= runeLen {
		return runeLen
	}

	maxAdjust := 10 // Maximum characters to scan for boundary

	if isEnd {
		// Look forward for separator
		for i := pos; i < runeLen && i < pos+maxAdjust; i++ {
			if e.isSeparator(runes[i]) {
				return i
			}
		}
	} else {
		// Look backward for separator
		for i := pos - 1; i >= 0 && i >= pos-maxAdjust; i-- {
			if e.isSeparator(runes[i]) {
				return i + 1 // Position after separator
			}
		}
	}

	return pos
}

// isSeparator returns true if the rune is a word separator.
func (e *SnippetExtractor) isSeparator(r rune) bool {
	// Common separators: whitespace, punctuation
	if unicode.IsSpace(r) {
		return true
	}

	// Chinese/English punctuation
	separators := []rune{
		'。', '，', '、', '；', '：', '！', '？', '…', // Chinese
		'.', ',', '!', '?', ';', ':', // English
	}
	for _, sep := range separators {
		if r == sep {
			return true
		}
	}
	return false
}

// buildSnippet constructs the final snippet string with optional ellipsis.
func (e *SnippetExtractor) buildSnippet(runes []rune, start, end int, addEllipsis bool) (string, int) {
	runeLen := len(runes)
	var builder strings.Builder
	prefixLen := 0

	// Add prefix ellipsis if not starting from beginning
	if addEllipsis && start > 0 {
		builder.WriteString("...")
		prefixLen = 3
	}

	// Add content
	builder.WriteString(string(runes[start:end]))

	// Add suffix ellipsis if not ending at content end
	if addEllipsis && end < runeLen {
		builder.WriteString("...")
	}

	return builder.String(), prefixLen
}

// adjustMatchPositions adjusts highlight positions relative to the snippet.
// Only includes matches that fall within the snippet window.
func (e *SnippetExtractor) adjustMatchPositions(
	matches []Highlight,
	windowStart, windowEnd, prefixLen int,
) []Highlight {
	adjusted := make([]Highlight, 0, len(matches))

	for _, m := range matches {
		// Only include matches fully within the window
		if m.Start >= windowStart && m.End <= windowEnd {
			adjusted = append(adjusted, Highlight{
				Start:       m.Start - windowStart + prefixLen,
				End:         m.End - windowStart + prefixLen,
				MatchedText: m.MatchedText,
			})
		}
	}

	return adjusted
}

// ExtractMultipleSnippets extracts snippets centered on different matches.
// Only generates snippets for matches that are sufficiently far apart to avoid overlap.
// Useful for showing multiple relevant excerpts from the same content.
func (e *SnippetExtractor) ExtractMultipleSnippets(
	content string,
	matches []Highlight,
	maxSnippets int,
	opts *ExtractOptions,
) []struct {
	Snippet    string
	Highlights []Highlight
} {
	if opts == nil {
		opts = DefaultExtractOptions()
	}

	if len(matches) == 0 {
		snippet, _ := e.ExtractSnippet(content, nil, opts)
		return []struct {
			Snippet    string
			Highlights []Highlight
		}{{Snippet: snippet, Highlights: nil}}
	}

	if maxSnippets <= 0 {
		maxSnippets = 1
	}

	// Filter matches to avoid overlapping snippets
	// Minimum distance between match centers to be considered distinct
	minDistance := opts.ContextChars * 2
	if minDistance <= 0 {
		minDistance = e.defaultContextChars * 2
	}

	distinctMatches := e.selectDistinctMatches(matches, minDistance, maxSnippets)

	results := make([]struct {
		Snippet    string
		Highlights []Highlight
	}, 0, len(distinctMatches))

	// Extract snippets centered on distinct matches
	for _, match := range distinctMatches {
		singleMatch := []Highlight{match}
		snippet, highlights := e.ExtractSnippet(content, singleMatch, opts)
		results = append(results, struct {
			Snippet    string
			Highlights []Highlight
		}{Snippet: snippet, Highlights: highlights})
	}

	return results
}

// selectDistinctMatches selects up to maxCount matches that are at least minDistance apart.
// This ensures multiple snippets show genuinely different context rather than overlapping content.
func (e *SnippetExtractor) selectDistinctMatches(matches []Highlight, minDistance, maxCount int) []Highlight {
	if len(matches) == 0 {
		return nil
	}

	selected := make([]Highlight, 0, maxCount)
	selected = append(selected, matches[0]) // Always include first match

	for i := 1; i < len(matches) && len(selected) < maxCount; i++ {
		// Check if this match is far enough from all selected matches
		isFarEnough := true
		for _, sel := range selected {
			distance := matches[i].Start - sel.Start
			if distance < 0 {
				distance = -distance
			}
			if distance < minDistance {
				isFarEnough = false
				break
			}
		}
		if isFarEnough {
			selected = append(selected, matches[i])
		}
	}

	return selected
}
