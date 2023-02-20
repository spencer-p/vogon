package ast

import (
	"fmt"
	"strings"
)

type TodoTxt struct {
	Groupings []Grouping `Newline* @@*`
}

type Grouping struct {
	Header   []string `("#" @( Text+ ) Newline)?`
	Children []*Entry `(@@ Newline?)*`
	//Block []Block `(@@ Newline)*`
}

//type SubGrouping struct {
//	Children []*Entry `(@@ Newline?)*`
//}

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
	SpecialTag *SpecialTag `| @Tag`
	Text       []string    `| (@Text)+`
}

type SpecialTag struct {
	Key   string
	Value string
}

func (s *SpecialTag) Capture(values []string) error {
	if len(values) != 1 {
		return fmt.Errorf("expected exactly 1 tag, got %d", len(values))
	}
	key, value, ok := strings.Cut(values[0], ":")
	if !ok {
		return fmt.Errorf("cannot cut %q with `:`", values[0])
	}
	s.Key = key
	s.Value = value
	return nil
}
