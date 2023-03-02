package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spencer-p/vogon/pkg/parse"
)

const (
	dataPath = "testdata"
)

type testFmtCase struct {
	name   string
	input  []byte
	output []byte
}

func TestFmt(t *testing.T) {
	table := make(map[string]*testFmtCase)

	err := filepath.WalkDir(dataPath, func(path string, d fs.DirEntry, err error) error {
		var (
			isInput  = strings.HasSuffix(path, ".input")
			isOutput = strings.HasSuffix(path, ".output")
		)
		if !(isInput || isOutput) {
			return nil
		}

		filename := filepath.Base(path)
		shortname := filename[:len(filename)-len(filepath.Ext(filename))]
		if _, ok := table[shortname]; !ok {
			table[shortname] = &testFmtCase{
				name: shortname,
			}
		}

		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to dump contents of %q: %w", path, err)
		}

		entry := table[shortname]
		if isInput {
			entry.input = contents
		}
		if isOutput {
			entry.output = contents
		}
		return nil
	})
	if err != nil {
		t.Errorf("Failed to load test data: %v", err)
	}

	now := time.Date(2022, time.January, 01, 0, 0, 0, 0, time.UTC)
	parser := parse.BuildParser()
	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			got := new(bytes.Buffer)
			err := Fmt(parser, now, got, tc.input)
			if err != nil {
				t.Errorf("unexpected error from Fmt: %v", err)
				return
			}
			if diff := cmp.Diff(got.String(), string(tc.output)); diff != "" {
				t.Errorf("Fmt() returned unexpected result (-got,+want):\n%s", diff)
			}
		})
	}
}

func FuzzFmt(f *testing.F) {
	if err := filepath.WalkDir(dataPath, func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, ".input") {
			return nil
		}

		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to dump contents of %q: %w", path, err)
		}

		f.Add(string(contents))

		return nil
	}); err != nil {
		f.Errorf("Failed to load test data: %v", err)
	}

	now := time.Date(2022, time.January, 01, 0, 0, 0, 0, time.UTC)
	parser := parse.BuildParser()
	f.Fuzz(func(t *testing.T, s string) {
		var first, second bytes.Buffer
		err := Fmt(parser, now, &first, []byte(s))
		if err != nil {
			t.Logf("skip because input is invalid: %v", err)
			return
		}
		if first.Len() == 0 {
			t.Logf("skip because result is empty: %v", err)
			return
		}

		// Now format a second time and look for irregularities.
		err = Fmt(parser, now, &second, first.Bytes())
		if err != nil {
			t.Errorf("second output was invalid: %v", err)
			return
		}

		if diff := cmp.Diff(first.String(), second.String()); diff != "" {
			t.Errorf("Fmt() not idempotent (-first,+second):\n%s", diff)
		}
	})
}
