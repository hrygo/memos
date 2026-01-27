package memo

import (
	"strings"
	"testing"
)

func TestSnippetExtractor_ExtractSnippet(t *testing.T) {
	extractor := NewSnippetExtractor()

	tests := []struct {
		name         string
		content      string
		matches      []Highlight
		contextChars int
		wantPrefix   string // Expected prefix (for partial match)
		wantSuffix   string // Expected suffix (for partial match)
		wantEllipsis bool   // Whether ellipsis should be present
	}{
		{
			name:         "No matches - returns beginning",
			content:      "This is a long piece of content that should be truncated at the beginning.",
			matches:      nil,
			contextChars: 20,
			wantPrefix:   "This is a long piece",
			wantEllipsis: true,
		},
		{
			name:    "Match at beginning",
			content: "Hello world, this is a test content with more text.",
			matches: []Highlight{
				{Start: 0, End: 5, MatchedText: "Hello"},
			},
			contextChars: 20,
			wantPrefix:   "Hello world",
			wantEllipsis: true,
		},
		{
			name:    "Match in middle",
			content: "The quick brown fox jumps over the lazy dog and runs away quickly.",
			matches: []Highlight{
				{Start: 16, End: 19, MatchedText: "fox"},
			},
			contextChars: 15,
			wantPrefix:   "...",
			wantSuffix:   "...",
		},
		{
			name:    "Match at end",
			content: "Start of content and then the important keyword.",
			matches: []Highlight{
				{Start: 40, End: 47, MatchedText: "keyword"},
			},
			contextChars: 20,
			wantPrefix:   "...",
			wantSuffix:   "keyword.",
		},
		{
			name:    "Multiple matches - uses first",
			content: "First match here and second match there.",
			matches: []Highlight{
				{Start: 6, End: 11, MatchedText: "match"},
				{Start: 26, End: 31, MatchedText: "match"},
			},
			contextChars: 15,
			wantPrefix:   "First match",
		},
		{
			name:         "Empty content",
			content:      "",
			matches:      nil,
			contextChars: 50,
			wantPrefix:   "",
		},
		{
			name:    "Chinese content",
			content: "这是一段中文内容，包含搜索关键词测试。",
			matches: []Highlight{
				{Start: 10, End: 13, MatchedText: "关键词"},
			},
			contextChars: 8,
			wantPrefix:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ExtractOptions{
				ContextChars: tt.contextChars,
				AddEllipsis:  true,
			}
			snippet, _ := extractor.ExtractSnippet(tt.content, tt.matches, opts)

			if tt.wantPrefix != "" && !strings.HasPrefix(snippet, tt.wantPrefix) {
				t.Errorf("Expected snippet to start with %q, got %q", tt.wantPrefix, snippet)
			}
			if tt.wantSuffix != "" && !strings.HasSuffix(snippet, tt.wantSuffix) {
				t.Errorf("Expected snippet to end with %q, got %q", tt.wantSuffix, snippet)
			}
			if tt.wantEllipsis && !strings.Contains(snippet, "...") {
				t.Errorf("Expected ellipsis in snippet, got %q", snippet)
			}
		})
	}
}

func TestSnippetExtractor_ExtractSnippet_HighlightPositions(t *testing.T) {
	extractor := NewSnippetExtractor()

	tests := []struct {
		name              string
		content           string
		matches           []Highlight
		contextChars      int
		wantHighlightText string // Text at adjusted highlight position
	}{
		{
			name:    "Highlight position adjusted for prefix ellipsis",
			content: "Some prefix text before the keyword appears here.",
			matches: []Highlight{
				{Start: 28, End: 35, MatchedText: "keyword"},
			},
			contextChars:      15,
			wantHighlightText: "keyword",
		},
		{
			name:    "Highlight at start - no adjustment needed",
			content: "keyword is at the very start of this content.",
			matches: []Highlight{
				{Start: 0, End: 7, MatchedText: "keyword"},
			},
			contextChars:      20,
			wantHighlightText: "keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ExtractOptions{
				ContextChars: tt.contextChars,
				AddEllipsis:  true,
			}
			snippet, highlights := extractor.ExtractSnippet(tt.content, tt.matches, opts)

			if len(highlights) == 0 {
				t.Fatalf("Expected at least one highlight, got none")
			}

			h := highlights[0]
			snippetRunes := []rune(snippet)

			if h.Start < 0 || h.End > len(snippetRunes) {
				t.Errorf("Highlight position out of bounds: start=%d, end=%d, snippet len=%d",
					h.Start, h.End, len(snippetRunes))
				return
			}

			extractedText := string(snippetRunes[h.Start:h.End])
			if extractedText != tt.wantHighlightText {
				t.Errorf("Highlight text mismatch: want %q, got %q", tt.wantHighlightText, extractedText)
			}
		})
	}
}

func TestSnippetExtractor_adjustToWordBoundary(t *testing.T) {
	extractor := NewSnippetExtractor()

	tests := []struct {
		name    string
		content string
		pos     int
		isEnd   bool
		wantPos int
	}{
		{
			name:    "Forward - finds space",
			content: "hello world test",
			pos:     7,
			isEnd:   true,
			wantPos: 11, // Finds space after "world"
		},
		{
			name:    "Forward - finds comma",
			content: "hello, world",
			pos:     4,
			isEnd:   true,
			wantPos: 5, // Position of comma
		},
		{
			name:    "Backward - finds space",
			content: "hello world test",
			pos:     8,
			isEnd:   false,
			wantPos: 6, // Position after "hello "
		},
		{
			name:    "Position at start",
			content: "hello",
			pos:     0,
			isEnd:   false,
			wantPos: 0,
		},
		{
			name:    "Position at end",
			content: "hello",
			pos:     5,
			isEnd:   true,
			wantPos: 5,
		},
		{
			name:    "Chinese - finds separator",
			content: "你好，世界",
			pos:     1,
			isEnd:   true,
			wantPos: 2, // Position of comma
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runes := []rune(tt.content)
			got := extractor.adjustToWordBoundary(runes, tt.pos, tt.isEnd)
			if got != tt.wantPos {
				t.Errorf("adjustToWordBoundary() = %d, want %d", got, tt.wantPos)
			}
		})
	}
}

func TestSnippetExtractor_calculateWindow(t *testing.T) {
	extractor := NewSnippetExtractor()

	tests := []struct {
		name         string
		center       int
		contentLen   int
		contextChars int
		wantStart    int
		wantEnd      int
	}{
		{
			name:         "Center in middle",
			center:       50,
			contentLen:   100,
			contextChars: 20,
			wantStart:    30,
			wantEnd:      70,
		},
		{
			name:         "Center near start - shift right",
			center:       5,
			contentLen:   100,
			contextChars: 20,
			wantStart:    0,
			wantEnd:      40,
		},
		{
			name:         "Center near end - shift left",
			center:       95,
			contentLen:   100,
			contextChars: 20,
			wantStart:    60,
			wantEnd:      100,
		},
		{
			name:         "Short content - full range",
			center:       10,
			contentLen:   20,
			contextChars: 30,
			wantStart:    0,
			wantEnd:      20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := extractor.calculateWindow(tt.center, tt.contentLen, tt.contextChars)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("calculateWindow() = (%d, %d), want (%d, %d)",
					start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestSnippetExtractor_ExtractMultipleSnippets(t *testing.T) {
	extractor := NewSnippetExtractor()

	content := "First keyword appears here. Second keyword is in the middle. Third keyword at the end."
	matches := []Highlight{
		{Start: 6, End: 13, MatchedText: "keyword"},
		{Start: 35, End: 42, MatchedText: "keyword"},
		{Start: 69, End: 76, MatchedText: "keyword"},
	}

	opts := &ExtractOptions{
		ContextChars: 15,
		AddEllipsis:  true,
	}

	results := extractor.ExtractMultipleSnippets(content, matches, 2, opts)

	if len(results) != 2 {
		t.Errorf("Expected 2 snippets, got %d", len(results))
	}

	// Each snippet should contain keyword
	for i, result := range results {
		if !strings.Contains(result.Snippet, "keyword") {
			t.Errorf("Snippet %d should contain 'keyword': %q", i, result.Snippet)
		}
	}
}

func TestSnippetExtractor_isSeparator(t *testing.T) {
	extractor := NewSnippetExtractor()

	tests := []struct {
		r    rune
		want bool
	}{
		{' ', true},
		{'\n', true},
		{'\t', true},
		{'。', true},
		{'，', true},
		{'.', true},
		{',', true},
		{'a', false},
		{'中', false},
		{'1', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.r), func(t *testing.T) {
			if got := extractor.isSeparator(tt.r); got != tt.want {
				t.Errorf("isSeparator(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

func TestSnippetExtractor_NoEllipsis(t *testing.T) {
	extractor := NewSnippetExtractor()

	content := "Short content only."
	opts := &ExtractOptions{
		ContextChars: 50,
		AddEllipsis:  false,
	}

	snippet, _ := extractor.ExtractSnippet(content, nil, opts)

	if strings.Contains(snippet, "...") {
		t.Errorf("Snippet should not contain ellipsis when AddEllipsis=false: %q", snippet)
	}
}

// Benchmark tests
func BenchmarkSnippetExtractor_ExtractSnippet(b *testing.B) {
	extractor := NewSnippetExtractor()

	content := strings.Repeat("This is a test sentence with some keywords. ", 100)
	matches := []Highlight{
		{Start: 500, End: 508, MatchedText: "keywords"},
	}
	opts := DefaultExtractOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.ExtractSnippet(content, matches, opts)
	}
}

func BenchmarkSnippetExtractor_ExtractMultipleSnippets(b *testing.B) {
	extractor := NewSnippetExtractor()

	content := strings.Repeat("This is a test sentence with some keywords. ", 100)
	matches := []Highlight{
		{Start: 100, End: 108, MatchedText: "keywords"},
		{Start: 500, End: 508, MatchedText: "keywords"},
		{Start: 1000, End: 1008, MatchedText: "keywords"},
	}
	opts := DefaultExtractOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.ExtractMultipleSnippets(content, matches, 3, opts)
	}
}
