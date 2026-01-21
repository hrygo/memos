package sqlite

import (
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
)

var (
	protojsonUnmarshaler = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
)

// placeholder returns a placeholder for SQLite (uses ?)
func placeholder(n int) string {
	return "?"
}

// placeholders returns n placeholders for SQLite
func placeholders(n int) string {
	list := []string{}
	for i := 0; i < n; i++ {
		list = append(list, placeholder(i+1))
	}
	return strings.Join(list, ", ")
}
