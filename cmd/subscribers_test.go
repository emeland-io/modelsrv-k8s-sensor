package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommaSeparatedList(t *testing.T) {
	urlA := "http://a:8080/api"
	urlB := "http://b:8080/api"
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "whitespace", raw: "  ,  ", want: nil},
		{name: "single", raw: "http://host:8080/api", want: []string{"http://host:8080/api"}},
		{name: "multiple", raw: urlA + "," + urlB, want: []string{urlA, urlB}},
		{name: "trimmed", raw: " " + urlA + " , " + urlB + " ", want: []string{urlA, urlB}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommaSeparatedList(tt.raw)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseCommaSeparatedListSkipsEmptySegments(t *testing.T) {
	got := parseCommaSeparatedList("http://a:8080/api,,http://b:8080/api")
	assert.Equal(t, []string{"http://a:8080/api", "http://b:8080/api"}, got)
}
