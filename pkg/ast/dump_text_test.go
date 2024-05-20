package ast

import (
	"bytes"
	"strings"
	"testing"
)

func ptr[T any](t T) *T {
	return &t
}

func TestDumpEntry(t *testing.T) {
	table := []struct {
		name string
		e    Entry
		want string
	}{{
		name: "simple",
		e: Entry{
			CreationDate: ptr("2024-01-01"),
			Description: []*DescriptionPart{{
				Text: []string{"hello", "world"},
			}},
		},
		want: "  2024-01-01 hello world\n",
	}, {
		name: "with notes",
		e: Entry{
			CreationDate: ptr("2024-01-01"),
			Description: []*DescriptionPart{{
				Text: []string{"This is a title of sorts"},
			}},
			Notes: []NoteLine{{
				Text: []string{"this", "is a note line 1"},
			}, {
				Text: []string{"and this is line 2"},
			}},
		},
		want: strings.Join([]string{
			"  2024-01-01 This is a title of sorts",
			"           | this is a note line 1",
			"           | and this is line 2",
		}, "\n") + "\n",
	}}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer
			_ = tc.e.DumpText(&b)
			got := b.String()
			if got != tc.want {
				t.Errorf("want did not match got:\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}
