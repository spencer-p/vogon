package parse

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spencer-p/vogon/pkg/ast"
)

func TestParse(t *testing.T) {
	table := []struct {
		name    string
		input   string
		wantErr bool
		assert  func(t *testing.T, result ast.TodoTxt)
	}{{
		name: "simple",
		input: `#Inbox
		2022-01-01 foo
		bar
		2022-01-01 2021-12-01 baz`,
	}, {
		name: "two dates",
		input: `#Inbox
		2022-01-01 2025-22-98 foo bar baz`,
	}, {
		name: "priority",
		input: `#Inbox
		A 2025-22-98 foo bar baz
		B qux and such`,
	}, {
		name: "tags and people",
		input: `#Inbox
		this is +tagged and has an @symbol`,
	}, {
		name:    "invalid header",
		input:   `#Inbox this is +tagged and has an @symbol`,
		wantErr: true,
	}, {
		name: "block spacing",
		input: `# Inbox
		this is one block
		it has two items

		this is the second block
		it has
		three items`,
		assert: func(t *testing.T, result ast.TodoTxt) {
			got := len(result.Groupings[0].Blocks)
			if want := 2; got != want {
				t.Errorf("wanted %d blocks, got %d", want, got)
			}
		},
	}}
	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			var result ast.TodoTxt
			p := BuildParser()
			err := p.ParseString("testinput", tc.input, &result)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("wantErr=%t, but got err=%q", tc.wantErr, err)
			}

			if tc.assert != nil {
				tc.assert(t, result)
			}

			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.SetIndent("", "  ")
			if err := enc.Encode(&result); err != nil {
				t.Errorf("failed to serialize result: %v", err)
			} else {
				t.Logf("%s", buf.Bytes())
			}
		})
	}
}
