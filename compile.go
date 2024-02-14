package main

import (
	"sort"
	"strings"

	"github.com/spencer-p/vogon/pkg/ast"
)

type HeaderCompiler struct {
	Header    string
	Filter    func(string, *ast.Entry) bool
	Transform func(*ast.Entry) *ast.Entry
	SortLess  func(l, r *ast.Entry) bool
}

func Compile(t ast.TodoTxt, compilers []HeaderCompiler) ast.TodoTxt {
	newEntries := make(map[string][]ast.Block)

	for _, grouping := range t.Groupings {
		origHeader := strings.Join(grouping.Header, " ")
		for blockNum, block := range grouping.Blocks {
			for _, e := range block.Children {
				insertBlock := blockNum
				dstHeader := origHeader
				var currentTransform func(*ast.Entry) *ast.Entry
				for _, compiler := range compilers {
					if compiler.Filter == nil {
						continue
					}
					if compiler.Filter(origHeader, e) {
						dstHeader = compiler.Header
						if compiler.Transform != nil {
							e = compiler.Transform(e)
						}
						if compiler.Header != origHeader {
							// If this entry is moving headers, put it in the
							// first block of the header.
							insertBlock = 0
						}
						goto insert // Already transformed, go straight to insert.
					} else if compiler.Header == origHeader {
						// Entry is already under correct header, but the header
						// did not pass its own filter. This entry may yet be
						// captured by another header - but if it doesn't, we
						// should hold on to the transformer so that we can
						// still apply it if needed.
						currentTransform = compiler.Transform
					}
				}

				// The entry is staying in its current header, but it did not pass
				// its own filter func. It will still get transformed according to
				// its header rules.
				if currentTransform != nil {
					e = currentTransform(e)
				}

			insert:
				for len(newEntries[dstHeader]) <= insertBlock {
					newEntries[dstHeader] = append(newEntries[dstHeader], ast.Block{})
				}
				newEntries[dstHeader][insertBlock].Children = append(newEntries[dstHeader][insertBlock].Children, e)
			}
		}
	}

	var result ast.TodoTxt
	sortLookup := sliceToMap(compilers, func(c HeaderCompiler) string { return c.Header })
	for header, blocks := range newEntries {
		if compiler, ok := sortLookup[header]; ok && compiler.SortLess != nil {
			for _, block := range blocks {
				sort.SliceStable(block.Children, func(i, j int) bool {
					left := block.Children[i]
					right := block.Children[j]
					return compiler.SortLess(left, right)
				})
			}

		}

		result.Groupings = append(result.Groupings, ast.Grouping{
			Header: []string{header}, // This may not be strictly correct, but the result is the same.
			Blocks: blocks,
		})
	}

	existingPriorities := map[string]int{}
	for i, g := range t.Groupings {
		existingPriorities[strings.Join(g.Header, " ")] = i
	}

	// Sort the groupings by desired order.
	headingPriority := map[string]int{
		"Inbox":     10,
		"Today":     20,
		"Evening":   21,
		"Scheduled": 30,
		"Next":      40,
		"Next week": 41,
		"Someday":   50,
		"Logged":    999,
	}
	sort.SliceStable(result.Groupings, func(i, j int) bool {
		leftHeader := strings.Join(result.Groupings[i].Header, " ")
		rightHeader := strings.Join(result.Groupings[j].Header, " ")
		left, leftKnown := headingPriority[leftHeader]
		right, rightKnown := headingPriority[rightHeader]

		// If either is not an official header, then attempt to preserve their
		// non-official ordering but leave them at the bottom, in between
		// Next and Someday.
		if !leftKnown || !rightKnown {
			if !leftKnown && !rightKnown {
				// If they are both unknown, then they both must be in
				// the existingPriorities map.
				return existingPriorities[leftHeader] < existingPriorities[rightHeader]
			} else if !leftKnown {
				return right >= headingPriority["Someday"]
			} else { // !rightKnown
				return left <= headingPriority["Next week"]
			}
		}

		return left < right
	})

	return result
}

func sliceToMap[T any](l []T, f func(T) string) map[string]T {
	result := make(map[string]T)
	for _, t := range l {
		key := f(t)
		result[key] = t
	}
	return result
}
