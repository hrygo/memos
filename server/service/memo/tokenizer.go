// Package memo provides memo-related services including search highlighting.
package memo

import (
	"strings"
	"unicode"
)

// Tokenizer handles text tokenization for search highlighting.
// Supports both Chinese and English text.
type Tokenizer struct {
	// minTokenLen is the minimum length for a token to be considered valid
	minTokenLen int
}

// NewTokenizer creates a new Tokenizer instance.
func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		minTokenLen: 1,
	}
}

// Tokenize splits the input text into searchable tokens.
// For Chinese: splits by character (each character is a token)
// For English: splits by whitespace and punctuation
func (t *Tokenizer) Tokenize(text string) []string {
	if text == "" {
		return nil
	}

	text = strings.TrimSpace(text)
	var tokens []string
	seen := make(map[string]bool)

	var currentWord strings.Builder
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			// Flush any pending English word
			if currentWord.Len() > 0 {
				word := strings.ToLower(currentWord.String())
				if len(word) >= t.minTokenLen && !seen[word] {
					tokens = append(tokens, word)
					seen[word] = true
				}
				currentWord.Reset()
			}
			// Add Chinese character as a token
			char := string(r)
			if !seen[char] {
				tokens = append(tokens, char)
				seen[char] = true
			}
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// Build English word
			currentWord.WriteRune(r)
		} else {
			// Flush on whitespace/punctuation
			if currentWord.Len() > 0 {
				word := strings.ToLower(currentWord.String())
				if len(word) >= t.minTokenLen && !seen[word] {
					tokens = append(tokens, word)
					seen[word] = true
				}
				currentWord.Reset()
			}
		}
	}

	// Flush remaining word
	if currentWord.Len() > 0 {
		word := strings.ToLower(currentWord.String())
		if len(word) >= t.minTokenLen && !seen[word] {
			tokens = append(tokens, word)
			seen[word] = true
		}
	}

	return tokens
}
