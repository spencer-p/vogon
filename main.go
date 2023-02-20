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

	"github.com/spencer-p/vogon/pkg/ast"
	"github.com/spencer-p/vogon/pkg/dates"
	"github.com/spencer-p/vogon/pkg/parse"

	"github.com/alecthomas/participle/v2"
)

const (
	dateFmt = "2006-01-02"
)

var (
	ebnf     = flag.Bool("ebnf", false, "Output EBNF")
	verbose  = flag.Bool("v", false, "Print more")
	filename = flag.String("f", "-", "todo.txt file path to process")
)

func main() {
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

	parser := parse.BuildParser()
	if *ebnf {
		fmt.Println(parser.String())
		return
	}

	rawInput, ok := <-rawInputCh
	if !ok {
		return
	}
	err := Fmt(parser, time.Now(), os.Stdout, rawInput)
	if err != nil {
		// If formatting failed, dump the original + an error.
		fmt.Fprintln(os.Stderr, err)
		os.Stderr.Write(rawInput)
		os.Exit(1)
	}

}

func Fmt(parser *participle.Parser, now time.Time, output io.Writer, input []byte) error {
	var t ast.TodoTxt
	if err := parser.ParseBytes("", input, &t); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	if *verbose {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&t); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	today := now.Format(dateFmt)
	visitAllEntries(&t, func(heading string, entry *ast.Entry) error {
		// Add creation dates.
		if entry.CreationDate == nil {
			entry.CreationDate = &today
		}
		return nil
	})

	t = Compile(t, []HeaderCompiler{{
		Header: "Logged",
		Filter: func(header string, e *ast.Entry) bool { return e.Completed == true },
		Transform: func(e *ast.Entry) *ast.Entry {
			e.Completed = true
			if e.CompletionDate == nil {
				e.CompletionDate = &today
			}
			return e
		},
		SortLess: func(l, r *ast.Entry) bool { return *l.CompletionDate > *r.CompletionDate },
	}, {
		Header: "Today",
		Filter: func(header string, e *ast.Entry) bool {
			scheduledFor, ok := e.ScheduledFor()
			if !ok {
				return false // No scheduled date.
			}
			// Accept "t", "today", and the formatted date for today.
			return scheduledFor == "t" || scheduledFor == "today" || maybeNormalizeDate(now, scheduledFor) <= today
		},
		Transform: func(e *ast.Entry) *ast.Entry {
			ast.SliceRemove(&(*e).Description, func(dp *ast.DescriptionPart) bool {
				return dp.SpecialTag != nil && ast.StringIsScheduled(dp.SpecialTag.Key)
			})
			return e
		},
		// No sorting for Today.
	}, manualHeader(
		"Next",
	), manualHeader(
		"Someday",
	), manualHeader(
		"Waiting",
	), {
		Header: "Scheduled",
		Filter: func(header string, e *ast.Entry) bool { _, ok := e.ScheduledFor(); return ok },
		SortLess: func(l, r *ast.Entry) bool {
			schedLeft, _ := l.ScheduledFor()
			schedRight, _ := r.ScheduledFor()
			return maybeNormalizeDate(now, schedLeft) < maybeNormalizeDate(now, schedRight)
		},
		Transform: func(e *ast.Entry) *ast.Entry {
			// Rewrite the scheduled date to canonical form instead of relative
			// form, if needed.
			for i := range e.Description {
				if e.Description[i].SpecialTag != nil && ast.StringIsScheduled(e.Description[i].SpecialTag.Key) {
					date := maybeNormalizeDate(now, e.Description[i].SpecialTag.Value)
					e.Description[i].SpecialTag.Value = date
				}
			}
			return e
		},
	}, {
		Header: "Inbox",
		Filter: func(header string, e *ast.Entry) bool { return header == "" },
	}, {
		Header:    "nop",
		Filter:    func(header string, e *ast.Entry) bool { return false },
		Transform: func(e *ast.Entry) *ast.Entry { return e },
		SortLess:  func(l, r *ast.Entry) bool { return false },
	}})

	bufOutput := bufio.NewWriter(output)
	err := t.DumpText(bufOutput)
	if err != nil {
		return fmt.Errorf("unable to format: %w", err)
	}
	return bufOutput.Flush()
}

func visitAllEntries(t *ast.TodoTxt, visit func(heading string, e *ast.Entry) error) error {
	for gi := range t.Groupings {
		heading := strings.Join(t.Groupings[gi].Header, " ")
		for bi := range t.Groupings[gi].Blocks {
			for ci := range t.Groupings[gi].Blocks[bi].Children {
				err := visit(heading, t.Groupings[gi].Blocks[bi].Children[ci])
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func findEntries(t *ast.TodoTxt, predicate func(heading string, e *ast.Entry) bool) []**ast.Entry {
	var result []**ast.Entry
	for gi := range t.Groupings {
		heading := strings.Join(t.Groupings[gi].Header, " ")
		for bi := range t.Groupings[gi].Blocks {
			for ci := range t.Groupings[gi].Blocks[bi].Children {
				e := &t.Groupings[gi].Blocks[bi].Children[ci]
				accept := predicate(heading, *e)
				if accept {
					result = append(result, e)
				}
			}
		}
	}
	return result
}

func findGrouping(t *ast.TodoTxt, name string) *ast.Grouping {
	for gi := range t.Groupings {
		if strings.Join(t.Groupings[gi].Header, " ") == name {
			return &t.Groupings[gi]
		}
	}

	t.Groupings = append(t.Groupings, ast.Grouping{
		Header: []string{name},
	})
	return &t.Groupings[len(t.Groupings)-1]
}

func maybeNormalizeDate(now time.Time, date string) string {
	if norm, err := dates.ParseRelative(now, date); err == nil {
		return norm.Format(dateFmt)
	}
	return date
}

func manualHeader(headerName string) HeaderCompiler {
	return HeaderCompiler{
		Header: headerName,
		Filter: func(header string, e *ast.Entry) bool {
			move, ok := e.Tag("move")
			if !ok {
				move, ok = e.ScheduledFor()
			}
			return ok && move == strings.ToLower(headerName)
		},
		Transform: func(e *ast.Entry) *ast.Entry { e.RemoveTag("move"); e.RemoveTag("sched"); return e },
	}
}
