package utils

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestEscapeMarkdown(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "asd",
			expected: "asd",
		},
		{
			input:    `cool~`,
			expected: `cool\~`,
		},
		{
			input:    `cool_`,
			expected: `cool\_`,
		},
		{
			input:    `cool_man_`,
			expected: `cool\_man\_`,
		},
		{
			input:    `**`,
			expected: `\*\*`,
		},
		{
			input:    `cool*`,
			expected: `cool\*`,
		},
		{
			input:    `||spoiler||`,
			expected: `\|\|spoiler\|\|`,
		},
		{
			input:    `*cool`,
			expected: `\*cool`,
		},
		{
			input:    `\\*`,
			expected: `\\\\\*`,
		},
		{
			input:    `cool\*`,
			expected: `cool\\\*`,
		},
		{
			input:    `cool\**`,
			expected: `cool\\\*\*`,
		},
		{
			input:    `****`,
			expected: `\*\*\*\*`,
		},
		{
			input:    `\****`,
			expected: `\\\*\*\*\*`,
		},
		{
			input:    "cool`",
			expected: "cool\\`",
		},
	}

	for _, test := range tests {
		actual := EscapeMarkdown(test.input)
		c.Assert(actual, qt.Equals, test.expected)
	}
}
