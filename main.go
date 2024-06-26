package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
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
		ReBlock:  blockByWeek,
	}, {
		Header: "Today",
		Filter: func(header string, e *ast.Entry) bool {
			dueDate, hasDueDate := e.DueDate()
			scheduledFor, hasScheduled := e.ScheduledFor()
			if !hasDueDate && !hasScheduled {
				return false // No scheduled or due date.
			}

			// Check for scheduled date.
			// Accept "t", "today", and the formatted date for today.
			norm, err := normalizeDate(now, scheduledFor)
			if scheduledFor == "t" || scheduledFor == "today" || (err == nil && norm <= today) {
				return true
			}

			// Check for due date.
			norm, err = normalizeDate(now, dueDate)
			if err == nil && norm <= today {
				return true
			}
			return false

		},
		Transform: func(e *ast.Entry) *ast.Entry {
			ast.SliceRemove(&(*e).Description, func(dp *ast.DescriptionPart) bool {
				return dp.SpecialTag != nil && ast.StringIsScheduled(dp.SpecialTag.Key)
			})
			return e
		},
		// No sorting for Today.
	}, manualHeader(
		"Next", now,
	), manualHeader(
		"Someday", now,
	), manualHeader(
		"Waiting", now,
	), manualHeader(
		"Evening", now,
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
		Transform: func(e *ast.Entry) *ast.Entry {
			normalizeDateTag(e, now, "due")
			return e
		},
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

func normalizeDate(now time.Time, date string) (string, error) {
	if norm, err := dates.ParseRelative(now, date); err == nil {
		return norm.Format(dateFmt), nil
	}
	if _, err := time.Parse(dateFmt, date); err == nil {
		return date, nil
	}
	return "", fmt.Errorf("date %q is not a relative date or YYYY-MM-DD", date)
}

func manualHeader(headerName string, now time.Time) HeaderCompiler {
	return HeaderCompiler{
		Header: headerName,
		Filter: func(header string, e *ast.Entry) bool {
			move, ok := e.Tag("move")
			if !ok {
				move, ok = e.ScheduledFor()
			}
			return ok && move == strings.ToLower(headerName)
		},
		Transform: func(e *ast.Entry) *ast.Entry {
			e.RemoveTag("move")
			e.RemoveTag("sched")
			e.RemoveTag("s")
			normalizeDateTag(e, now, "due")
			return e
		},
	}
}

func normalizeDateTag(e *ast.Entry, now time.Time, tags ...string) {
	tagLookup := make(map[string]bool)
	for _, t := range tags {
		tagLookup[t] = true
	}

	for i := range e.Description {
		if e.Description[i].SpecialTag != nil && tagLookup[e.Description[i].SpecialTag.Key] {
			date := maybeNormalizeDate(now, e.Description[i].SpecialTag.Value)
			e.Description[i].SpecialTag.Value = date
		}
	}
}

func partition[T any](l []T, f func(T) bool) [][]T {
	head := make([]T, 0)
	tail := make([]T, 0)
	for _, li := range l {
		if f(li) {
			head = append(head, li)
		} else {
			tail = append(tail, li)
		}
	}
	return [][]T{head, tail}
}

// blockByWeek splits the first block based on the week it was completed in.
// It only splits the first week to avoid sorting an entire logbook.
// The first block is only split in two. The first new block may have
// completion dates spread over multiple weeks. The second new block is
// guaranteed to be only items completed in a single week (the oldest week
// present in the initial first block).
// Repeating this process converges on fully split weeks.
func blockByWeek(blocks []ast.Block) []ast.Block {
	if len(blocks) == 0 {
		return blocks
	}
	if len(blocks[0].Children) == 0 {
		return blocks
	}

	firstblock := blocks[0]
	minweek := firstblock.Children[0].CompletedWeek()
	for _, e := range firstblock.Children {
		if week := e.CompletedWeek(); week < minweek {
			minweek = week
		}
	}
	split := partition(firstblock.Children, func(e *ast.Entry) bool {
		return e.CompletedWeek() > minweek
	})
	return slices.Replace(blocks, 0, 1,
		ast.Block{Children: split[0]},
		ast.Block{Children: split[1]},
	)
}
