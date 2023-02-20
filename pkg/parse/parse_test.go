package parse

import (
	"testing"

	"github.com/spencer-p/vogon/pkg/ast"
)

func TestParse(t *testing.T) {
	table := []struct {
		name    string
		input   string
		wantErr bool
	}{{
		name: "simple",
		input: `#Inbox
		2022-01-01 foo bar baz`,
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
		input: `#Inbox
		this is one block
		it has two items
		
		this is the second block
		it has
		three items`,
		wantErr: false,
	}}
	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			var result ast.TodoTxt
			p := BuildParser()
			err := p.ParseString("testinput", tc.input, &result)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("wantErr=%t, but got err=%q", tc.wantErr, err)
			}
		})
	}
}
