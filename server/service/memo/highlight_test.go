package memo

import (
	"testing"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tokenizer := NewTokenizer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "English words",
			input:    "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "Chinese characters",
			input:    "你好世界",
			expected: []string{"你", "好", "世", "界"},
		},
		{
			name:     "Mixed Chinese and English",
			input:    "Go语言",
			expected: []string{"go", "语", "言"},
		},
		{
			name:     "With punctuation",
			input:    "Hello, World!",
			expected: []string{"hello", "world"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "Duplicate words",
			input:    "test test TEST",
			expected: []string{"test"},
		},
		{
			name:     "Numbers",
			input:    "test123 456",
			expected: []string{"test123", "456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Tokenize(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Tokenize(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}

			for i, token := range result {
				if token != tt.expected[i] {
					t.Errorf("Tokenize(%q)[%d] = %q, want %q", tt.input, i, token, tt.expected[i])
				}
			}
		})
	}
}

func TestHighlightService_findMatches(t *testing.T) {
	service := &HighlightService{
		tokenizer: NewTokenizer(),
	}

	tests := []struct {
		name          string
		content       string
		tokens        []string
		expectedCount int
	}{
		{
			name:          "Single match",
			content:       "Hello World",
			tokens:        []string{"hello"},
			expectedCount: 1,
		},
		{
			name:          "Multiple matches",
			content:       "Go is great. Go is fast.",
			tokens:        []string{"go"},
			expectedCount: 2,
		},
		{
			name:          "Case insensitive",
			content:       "GO Go go",
			tokens:        []string{"go"},
			expectedCount: 3,
		},
		{
			name:          "Chinese match",
			content:       "今天完成了项目评审",
			tokens:        []string{"项", "目"},
			expectedCount: 2,
		},
		{
			name:          "No matches",
			content:       "Hello World",
			tokens:        []string{"xyz"},
			expectedCount: 0,
		},
		{
			name:          "Empty tokens",
			content:       "Hello World",
			tokens:        []string{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := service.findMatches(tt.content, tt.tokens)
			if len(matches) != tt.expectedCount {
				t.Errorf("findMatches() returned %d matches, want %d", len(matches), tt.expectedCount)
			}
		})
	}
}

func TestHighlightService_extractSnippet(t *testing.T) {
	extractor := NewSnippetExtractor()

	tests := []struct {
		name         string
		content      string
		matches      []Highlight
		contextChars int
		wantPrefix   bool
		wantSuffix   bool
	}{
		{
			name:         "No matches - returns beginning",
			content:      "This is a very long content that should be truncated",
			matches:      nil,
			contextChars: 10,
			wantPrefix:   false,
			wantSuffix:   true,
		},
		{
			name:    "Match at beginning",
			content: "Hello World and more content here with additional text to make it longer",
			matches: []Highlight{
				{Start: 0, End: 5, MatchedText: "Hello"},
			},
			contextChars: 20,
			wantPrefix:   false,
			wantSuffix:   true,
		},
		{
			name:    "Match in middle",
			content: "This is some text with a match in the middle of the content",
			matches: []Highlight{
				{Start: 25, End: 30, MatchedText: "match"},
			},
			contextChars: 15,
			wantPrefix:   true,
			wantSuffix:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ExtractOptions{ContextChars: tt.contextChars, AddEllipsis: true}
			snippet, _ := extractor.ExtractSnippet(tt.content, tt.matches, opts)

			hasPrefix := len(snippet) >= 3 && snippet[:3] == "..."
			hasSuffix := len(snippet) >= 3 && snippet[len(snippet)-3:] == "..."

			if hasPrefix != tt.wantPrefix {
				t.Errorf("extractSnippet() prefix = %v, want %v, snippet = %q", hasPrefix, tt.wantPrefix, snippet)
			}
			if hasSuffix != tt.wantSuffix {
				t.Errorf("extractSnippet() suffix = %v, want %v, snippet = %q", hasSuffix, tt.wantSuffix, snippet)
			}
		})
	}
}

func TestHighlightService_removeOverlaps(t *testing.T) {
	service := &HighlightService{}

	tests := []struct {
		name     string
		matches  []Highlight
		expected int
	}{
		{
			name:     "No overlaps",
			matches:  []Highlight{{Start: 0, End: 5}, {Start: 10, End: 15}},
			expected: 2,
		},
		{
			name:     "With overlaps",
			matches:  []Highlight{{Start: 0, End: 5}, {Start: 3, End: 8}},
			expected: 1,
		},
		{
			name:     "Adjacent (no overlap)",
			matches:  []Highlight{{Start: 0, End: 5}, {Start: 5, End: 10}},
			expected: 2,
		},
		{
			name:     "Empty",
			matches:  []Highlight{},
			expected: 0,
		},
		{
			name:     "Single",
			matches:  []Highlight{{Start: 0, End: 5}},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.removeOverlaps(tt.matches)
			if len(result) != tt.expected {
				t.Errorf("removeOverlaps() = %d matches, want %d", len(result), tt.expected)
			}
		})
	}
}
