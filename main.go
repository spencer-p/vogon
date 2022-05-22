package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type TodoTxt struct {
	Groupings []Grouping `@@*`
}

type Grouping struct {
	Header   []string `("#" @( Text+ ) Newline)?`
	Children []Entry  `(@@ Newline?)*`
	// SubGroupings []SubGrouping `@@*`
}

// type SubGrouping struct {
// 	Header   []string `"#" "#" @( Text+ ) Newline`
// 	Children []Entry  `(@@ Newline?)*`
// }

type Entry struct {
	Completed      bool               `@"x"?`
	Priority       *string            `@Priority?`
	CompletionDate *string            `(@Date`
	CreationDate   *string            ` @Date | @Date)?`
	Description    []*DescriptionPart `@@*`
}

type DescriptionPart struct {
	Project    *string     `  "+"@Text`
	Context    *string     `| "@"@Text`
	SpecialTag *SpecialTag `| @@`
	Text       []string    `| @Text+`
}

type SpecialTag struct {
	Key   string `@Text`
	Value string `":" @Text`
}

func (e *Entry) MarshalText_() ([]byte, error) {
	buf := new(bytes.Buffer)
	if e.Completed {
		buf.WriteByte('x')
	} else {
		buf.WriteByte(' ')
	}
	if e.Priority != nil {
		fmt.Fprintf(buf, " %s", *e.Priority)
	}
	if e.CompletionDate != nil {
		fmt.Fprintf(buf, " %s", *e.CompletionDate)
	}
	if e.CreationDate != nil {
		fmt.Fprintf(buf, " %s", *e.CreationDate)
	}
	for _, p := range e.Description {
		buf.WriteByte(' ')
		switch {
		case len(p.Text) != 0:
			for i := range p.Text {
				if i != 0 {
					buf.WriteByte(' ')
				}
				buf.WriteString(p.Text[i])
			}
		case p.Context != nil:
			buf.WriteByte('@')
			buf.WriteString(*p.Context)
		case p.Project != nil:
			buf.WriteByte('+')
			buf.WriteString(*p.Project)
		case p.SpecialTag != nil:
			buf.WriteString(p.SpecialTag.Key)
			buf.WriteByte(':')
			buf.WriteString(p.SpecialTag.Value)
		}
	}
	return buf.Bytes(), nil
}

func main() {
	ebnf := flag.Bool("ebnf", false, "Output EBNF")
	verbose := flag.Bool("v", false, "Print more")
	filename := flag.String("f", "-", "todo.txt file path to process")
	flag.Parse()

	buf := new(bytes.Buffer)
	waitCh := make(chan struct{})
	go func() {
		defer func() {
			waitCh <- struct {
			}{}
		}()

		var input io.ReadCloser
		if *filename == "-" {
			input = os.Stdin
		} else {
			var err error
			input, err = os.Open(*filename)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
		defer input.Close()

		io.Copy(buf, input)
	}()

	lex := lexer.MustSimple([]lexer.SimpleRule{{
		Name:    "Date",
		Pattern: `\d{4}-\d{2}-\d{2}`,
	}, {
		Name:    "Priority",
		Pattern: `\(([A-Z])\)`,
	}, {
		Name:    "Punct",
		Pattern: `[\+@:#]`,
	}, {
		Name:    "Space",
		Pattern: ` `,
	}, {
		Name:    "Newline",
		Pattern: `\n+`,
	}, {
		Name:    "Text",
		Pattern: `[^\s][^:\s]*`,
	}})
	parser := participle.MustBuild(&TodoTxt{},
		participle.Lexer(lex),
		participle.Elide("Space"))
	if *ebnf {
		fmt.Println(parser.String())
		return
	}

	<-waitCh
	raw := buf.Bytes()
	var t TodoTxt
	if err := parser.ParseBytes(*filename, raw, &t); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *verbose {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&t); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	for i, g := range t.Groupings {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("# %s\n\n", strings.Join(g.Header, " "))
		for _, e := range g.Children {
			line, _ := e.MarshalText_()
			fmt.Printf("%s\n", line)
		}
	}
}
