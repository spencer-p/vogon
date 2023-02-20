package ast

import (
	"fmt"
	"io"
	"strings"
)

func (e *Entry) DumpText(out io.Writer) error {
	if e == nil {
		return nil
	}

	if e.Completed {
		out.Write([]byte{'x'})
	} else {
		out.Write([]byte{' '})
	}
	if e.Priority != nil {
		fmt.Fprintf(out, " %s", *e.Priority)
	}
	if e.CompletionDate != nil {
		fmt.Fprintf(out, " %s", *e.CompletionDate)
	}
	if e.CreationDate != nil {
		fmt.Fprintf(out, " %s", *e.CreationDate)
	}
	for _, p := range e.Description {
		out.Write([]byte{' '})
		switch {
		case len(p.Text) != 0:
			for i := range p.Text {
				if i != 0 {
					out.Write([]byte{' '})
				}
				out.Write([]byte(p.Text[i]))
			}
		case p.Context != nil:
			out.Write([]byte{'@'})
			out.Write([]byte(*p.Context))
		case p.Project != nil:
			out.Write([]byte{'+'})
			out.Write([]byte(*p.Project))
		case p.SpecialTag != nil:
			out.Write([]byte(p.SpecialTag.Key))
			out.Write([]byte{':'})
			out.Write([]byte(p.SpecialTag.Value))
		}
	}
	fmt.Fprintln(out)
	return nil
}

func (t TodoTxt) DumpText(out io.Writer) error {
	skipped := 0
	for i, g := range t.Groupings {
		if g.Len() == 0 {
			skipped++
			continue
		}
		if i-skipped > 0 {
			fmt.Fprintln(out)
		}
		if i-skipped == 0 && len(g.Header) == 0 {
			g.Header = []string{"Inbox"}
		}
		fmt.Fprintf(out, "# %s\n\n", strings.Join(g.Header, " "))
		for blockNum, b := range g.Blocks {
			for _, e := range b.Children {
				if e == nil {
					continue
				}
				err := e.DumpText(out)
				if err != nil {
					return err
				}
			}

			if blockNum+1 < len(g.Blocks) {
				fmt.Fprintf(out, "\n")
			}
		}
	}

	return nil
}
