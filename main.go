package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

const (
	dateFmt = "2006-01-02"
)

type TodoTxt struct {
	Groupings []Grouping `Newline* @@*`
}

type Grouping struct {
	Header   []string `("#" @( Text+ ) Newline)?`
	Children []*Entry `(@@ Newline?)*`
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
	Text       []string    `| (@Text (?! ":"))+`
}

type SpecialTag struct {
	Key   string `@Text`
	Value string `":" @Text`
}

func (e *Entry) MarshalText_() ([]byte, error) {
	if e == nil {
		return []byte{}, nil
	}

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

func (t TodoTxt) MarshalText_() ([]byte, error) {
	var buf bytes.Buffer

	for i, g := range t.Groupings {
		if len(g.Children) == 0 {
			// Not an exhaustive check; there may be many nil-valued children.
			continue
		}
		if i > 0 {
			fmt.Fprintln(&buf)
		}
		if i == 0 && len(g.Header) == 0 {
			g.Header = []string{"Inbox"}
		}
		fmt.Fprintf(&buf, "# %s\n\n", strings.Join(g.Header, " "))
		for _, e := range g.Children {
			if e == nil {
				continue
			}
			line, err := e.MarshalText_()
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&buf, "%s\n", line)
		}
	}

	return buf.Bytes(), nil
}

func main() {
	ebnf := flag.Bool("ebnf", false, "Output EBNF")
	verbose := flag.Bool("v", false, "Print more")
	filename := flag.String("f", "-", "todo.txt file path to process")
	flag.Parse()

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

	parser := BuildParser()
	if *ebnf {
		fmt.Println(parser.String())
		return
	}

	var t TodoTxt
	err := Fmt(parser, time.Now(), os.Stdout, input)
	if err != nil {
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
}

func Fmt(parser *participle.Parser, now time.Time, output io.Writer, input io.Reader) error {
	var t TodoTxt
	if err := parser.Parse("", input, &t); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Formatting goes here.
	// Add creation dates.
	// If completed, add completion date.
	// Move completed to Logged with completion date.
	// Move sched: to Scheduled.
	// Move sched:today to Today.
	// Sort Logged by completion date.
	// Sort Scheduled by sched: date.
	today := now.Format(dateFmt)
	visitAllEntries(&t, func(heading string, entry *Entry) error {
		// Add creation and completion dates.
		if entry.CreationDate == nil {
			entry.CreationDate = &today
		}
		if heading == "Logged" {
			entry.Completed = true
		}
		if entry.Completed && entry.CompletionDate == nil {
			entry.CompletionDate = &today
		}
		return nil
	})

	// Move completed to Logged.
	logged := findGrouping(&t, "Logged")
	completed := findEntries(&t, func(heading string, entry *Entry) bool {
		return entry.Completed && heading != "Logged"
	})
	for _, pe := range completed {
		logged.Children = append(logged.Children, *pe)
		*pe = nil
	}

	// Sort Logged.
	sort.SliceStable(logged.Children, func(i, j int) bool {
		// Sort nil or non-completed first. Noncompleted should not be in the Logged
		// suggestion. Required to tolerate bad input.
		if logged.Children[i] == nil || logged.Children[i].Completed == false {
			return true
		}
		if logged.Children[j] == nil || logged.Children[j].Completed == false {
			return false
		}

		return *logged.Children[i].CompletionDate > *logged.Children[j].CompletionDate
	})

	result, err := t.MarshalText_()
	if err != nil {
		return fmt.Errorf("unable to format: %w", err)
	}
	_, err = output.Write(result)
	return err
}

func BuildParser() *participle.Parser {
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
		participle.Elide("Space"),
		participle.UseLookahead(1))
	return parser
}

func visitAllEntries(t *TodoTxt, visit func(heading string, e *Entry) error) error {
	for gi := range t.Groupings {
		heading := strings.Join(t.Groupings[gi].Header, " ")
		for ci := range t.Groupings[gi].Children {
			err := visit(heading, t.Groupings[gi].Children[ci])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func findEntries(t *TodoTxt, predicate func(heading string, e *Entry) bool) []**Entry {
	var result []**Entry
	for gi := range t.Groupings {
		heading := strings.Join(t.Groupings[gi].Header, " ")
		for ci := range t.Groupings[gi].Children {
			e := &t.Groupings[gi].Children[ci]
			accept := predicate(heading, *e)
			if accept {
				result = append(result, e)
			}
		}
	}
	return result
}

func findGrouping(t *TodoTxt, name string) *Grouping {
	for gi := range t.Groupings {
		if strings.Join(t.Groupings[gi].Header, " ") == name {
			return &t.Groupings[gi]
		}
	}

	t.Groupings = append(t.Groupings, Grouping{
		Header: []string{name},
	})
	return &t.Groupings[len(t.Groupings)-1]
}
