package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/spencer-p/vogon/pkg/dates"
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
	Header         string
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
	Value string `":" (@Text | @Date)`
}

func main() {
	ebnf := flag.Bool("ebnf", false, "Output EBNF")
	verbose := flag.Bool("v", false, "Print more")
	filename := flag.String("f", "-", "todo.txt file path to process")
	flag.Parse()

	rawInputCh := make(chan []byte)
	go func() {
		defer close(rawInputCh)
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

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, input); err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", *filename, err)
			return
		}
		rawInputCh <- buf.Bytes()
	}()

	parser := BuildParser()
	if *ebnf {
		fmt.Println(parser.String())
		return
	}

	rawInput, ok := <-rawInputCh
	if !ok {
		return
	}
	var t TodoTxt
	err := Fmt(parser, time.Now(), os.Stdout, rawInput)
	if err != nil {
		// If formatting failed, dump the original + an error.
		fmt.Fprintln(os.Stderr, err)
		os.Stderr.Write(rawInput)
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

func Fmt(parser *participle.Parser, now time.Time, output io.Writer, input []byte) error {
	var t TodoTxt
	if err := parser.ParseBytes("", input, &t); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	today := now.Format(dateFmt)
	visitAllEntries(&t, func(heading string, entry *Entry) error {
		// Add creation dates.
		if entry.CreationDate == nil {
			entry.CreationDate = &today
		}
		return nil
	})

	t = t.Compile([]HeaderCompiler{{
		Header: "Logged",
		Filter: func(header string, e *Entry) bool { return e.Completed == true },
		Transform: func(e *Entry) *Entry {
			e.Completed = true
			if e.CompletionDate == nil {
				e.CompletionDate = &today
			}
			return e
		},
		SortLess: func(l, r *Entry) bool { return *l.CompletionDate > *r.CompletionDate },
	}, {
		Header: "Today",
		Filter: func(header string, e *Entry) bool {
			scheduledFor, ok := e.ScheduledFor()
			if !ok {
				return false // No scheduled date.
			}
			// Accept "t", "today", and the formatted date for today.
			return scheduledFor == "t" || scheduledFor == "today" || maybeNormalizeDate(now, scheduledFor) <= today
		},
		Transform: func(e *Entry) *Entry {
			sliceRemove(&(*e).Description, func(dp *DescriptionPart) bool {
				return dp.SpecialTag != nil && stringIsScheduled(dp.SpecialTag.Key)
			})
			return e
		},
		// No sorting for Today.
	}, {
		Header: "Scheduled",
		Filter: func(header string, e *Entry) bool { _, ok := e.ScheduledFor(); return ok },
		SortLess: func(l, r *Entry) bool {
			schedLeft, _ := l.ScheduledFor()
			schedRight, _ := r.ScheduledFor()
			return maybeNormalizeDate(now, schedLeft) < maybeNormalizeDate(now, schedRight)
		},
	}, {
		Header: "Inbox",
		Filter: func(header string, e *Entry) bool { return header == "" },
	}, {
		Header:    "nop",
		Filter:    func(header string, e *Entry) bool { return false },
		Transform: func(e *Entry) *Entry { return e },
		SortLess:  func(l, r *Entry) bool { return false },
	}})

	bufOutput := bufio.NewWriter(output)
	err := t.DumpText(bufOutput)
	if err != nil {
		return fmt.Errorf("unable to format: %w", err)
	}
	return bufOutput.Flush()
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

func sliceRemove[T any](s *[]T, filter func(T) bool) {
	result := make([]T, 0, len(*s)/2)
	for i := range *s {
		if filter((*s)[i]) {
			continue // Removed.
		}
		result = append(result, (*s)[i])
	}
	*s = result
}

func maybeNormalizeDate(now time.Time, date string) string {
	if norm, err := dates.ParseRelative(now, date); err == nil {
		return norm.Format(dateFmt)
	}
	return date
}
