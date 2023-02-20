package parse

import (
	"github.com/spencer-p/vogon/pkg/ast"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func BuildParser() *participle.Parser {
	lex := lexer.MustSimple([]lexer.SimpleRule{{
		Name:    "Date",
		Pattern: `\d{4}-\d{2}-\d{2}`,
	}, {
		Name:    "Priority",
		Pattern: `\(([A-Z])\)`,
	}, {
		Name:    "Punct",
		Pattern: `[\+@#]`,
	}, {
		Name:    "Tag",
		Pattern: `[^\s]+:[^:\s]+`,
	}, {
		Name:    "Space",
		Pattern: `( |\t)`,
	}, {
		Name:    "Newline",
		Pattern: `\n`,
	}, {
		Name:    "Text",
		Pattern: `[^\s]+`,
	}})
	parser := participle.MustBuild(&ast.TodoTxt{},
		participle.Lexer(lex),
		participle.Elide("Space"),
		participle.UseLookahead(1))
	return parser
}
