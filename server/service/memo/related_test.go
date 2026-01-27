package memo

import (
	"testing"
)

func TestExtractTagsFromContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "Single tag",
			content: "This is a note with #tag",
			want:    []string{"tag"},
		},
		{
			name:    "Multiple tags",
			content: "Note with #tag1 and #tag2",
			want:    []string{"tag1", "tag2"},
		},
		{
			name:    "Tag with punctuation",
			content: "Note with #tag, and more",
			want:    []string{"tag"},
		},
		{
			name:    "No tags",
			content: "Just a plain note",
			want:    nil,
		},
		{
			name:    "Duplicate tags",
			content: "#tag and #tag again",
			want:    []string{"tag"},
		},
		{
			name:    "Hash without word",
			content: "Just # symbol",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTagsFromContent(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("extractTagsFromContent() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractTagsFromContent()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIntersectTags(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{
			name: "Common tags",
			a:    []string{"tag1", "tag2", "tag3"},
			b:    []string{"tag2", "tag3", "tag4"},
			want: []string{"tag2", "tag3"},
		},
		{
			name: "No common tags",
			a:    []string{"tag1"},
			b:    []string{"tag2"},
			want: nil,
		},
		{
			name: "Empty first",
			a:    nil,
			b:    []string{"tag1"},
			want: nil,
		},
		{
			name: "Empty second",
			a:    []string{"tag1"},
			b:    nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intersectTags(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Errorf("intersectTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTitleFromContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "Single line",
			content: "This is the title",
			want:    "This is the title",
		},
		{
			name:    "Multi line",
			content: "Title here\nBody content",
			want:    "Title here",
		},
		{
			name:    "Long title",
			content: "This is a very long title that should be truncated to fit within the limit",
			want:    "This is a very long title that should be trunca...",
		},
		{
			name:    "Empty",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitleFromContent(tt.content)
			if got != tt.want {
				t.Errorf("extractTitleFromContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalculateContentSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		content1 string
		content2 string
		minScore float32
		maxScore float32
	}{
		{
			name:     "Identical content",
			content1: "Hello world",
			content2: "Hello world",
			minScore: 0.99,
			maxScore: 1.01,
		},
		{
			name:     "Partial overlap",
			content1: "Hello world foo",
			content2: "Hello world bar",
			minScore: 0.3,
			maxScore: 0.7,
		},
		{
			name:     "No overlap",
			content1: "Apple banana",
			content2: "Cat dog",
			minScore: 0,
			maxScore: 0.01,
		},
		{
			name:     "Empty content",
			content1: "",
			content2: "Hello",
			minScore: 0,
			maxScore: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateContentSimilarity(tt.content1, tt.content2)
			if got < tt.minScore || got > tt.maxScore {
				t.Errorf("calculateContentSimilarity() = %v, want between %v and %v",
					got, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestTokenizeForSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		content string
		minLen  int
	}{
		{
			name:    "English words",
			content: "Hello world test",
			minLen:  3,
		},
		{
			name:    "Chinese characters",
			content: "你好世界",
			minLen:  1, // Each CJK char is a token
		},
		{
			name:    "Mixed content",
			content: "Hello 你好 world 世界",
			minLen:  2,
		},
		{
			name:    "Single chars ignored",
			content: "a b c hello",
			minLen:  1, // Only "hello"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizeForSimilarity(tt.content)
			if len(got) < tt.minLen {
				t.Errorf("tokenizeForSimilarity() returned %d tokens, want at least %d",
					len(got), tt.minLen)
			}
		})
	}
}

func TestAbs64(t *testing.T) {
	tests := []struct {
		input int64
		want  int64
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
	}

	for _, tt := range tests {
		got := abs64(tt.input)
		if got != tt.want {
			t.Errorf("abs64(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
