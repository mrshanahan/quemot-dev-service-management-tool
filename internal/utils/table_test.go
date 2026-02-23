package utils

import (
	"strings"
	"testing"
)

func TestBuildComparisonTable(t *testing.T) {
	cases := []struct {
		hx       string
		hy       string
		xs       []string
		ys       []string
		expected string
	}{
		{
			"HEADER1",
			"HEADER2",
			[]string{"FOO", "BAR"},
			[]string{"FOO", "BANG"},
			`
     HEADER1 HEADER2
     ------- -------
BANG            X   
BAR     X           
FOO     X       X   
`,
		},
		{
			"HEADER1",
			"HEADER2",
			[]string{"FOO", "BAR"},
			[]string{"FOO", "BAR"},
			`
    HEADER1 HEADER2
    ------- -------
BAR    X       X   
FOO    X       X   
`,
		},
		{
			"HEADER1",
			"HEADER2",
			[]string{},
			[]string{},
			`
 HEADER1 HEADER2
 ------- -------
`,
		},
		{
			"HEADER1",
			"HEADER2",
			[]string{"FOO"},
			[]string{"FOO", "BAR", "ZING", "ZANG"},
			`
     HEADER1 HEADER2
     ------- -------
BAR             X   
FOO     X       X   
ZANG            X   
ZING            X   
`,
		},
		{
			"HEADER1",
			"HEADER2",
			[]string{"FOO", "BAR", "ZING", "ZANG"},
			[]string{"FOO"},
			`
     HEADER1 HEADER2
     ------- -------
BAR     X           
FOO     X       X   
ZANG    X           
ZING    X           
`,
		},
		{
			"HEADER1",
			"HEADER2",
			[]string{},
			[]string{"FOO"},
			`
    HEADER1 HEADER2
    ------- -------
FOO            X   
`,
		},
		{
			"HEADER1",
			"HEADER2",
			[]string{"FOO"},
			[]string{},
			`
    HEADER1 HEADER2
    ------- -------
FOO    X           
`,
		},
	}

	for _, c := range cases {
		t.Run("", func(s *testing.T) {
			s.Parallel()
			actual := BuildComparisonTable(c.hx, c.xs, c.hy, c.ys)
			trueExpected := strings.TrimLeft(c.expected, "\n")
			if actual != trueExpected {
				s.Errorf("results not equal; got:\n%s\nexpected:\n%s", actual, trueExpected)
			}
		})
	}
}
