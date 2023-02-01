package main

import (
	"sort"
	"strings"
)

type HeaderCompiler struct {
	Header    string
	Filter    func(string, *Entry) bool
	Transform func(*Entry) *Entry
	SortLess  func(l, r *Entry) bool
}

func (t TodoTxt) Compile(compilers []HeaderCompiler) TodoTxt {
	newEntries := make(map[string][]*Entry)

	for _, grouping := range t.Groupings {
		origHeader := strings.Join(grouping.Header, " ")
		for _, e := range grouping.Children {
			dstHeader := origHeader
			var currentTransform func(*Entry) *Entry
			for _, compiler := range compilers {
				if compiler.Filter == nil {
					continue
				}
				if compiler.Filter(origHeader, e) {
					dstHeader = compiler.Header
					if compiler.Transform != nil {
						e = compiler.Transform(e)
					}
					goto insert // Already transformed and sorted, go straight to insert.
				} else if compiler.Header == origHeader {
					// This is the transform function for if the entry stays in
					// its current header.
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
			newEntries[dstHeader] = append(newEntries[dstHeader], e)
		}
	}

	var result TodoTxt
	for header, children := range newEntries {
		for _, compiler := range compilers {
			if compiler.Header == header && compiler.SortLess != nil {
				sort.SliceStable(children, func(i, j int) bool {
					left := children[i]
					right := children[j]
					return compiler.SortLess(left, right)
				})
			}
		}

		result.Groupings = append(result.Groupings, Grouping{
			Header:   []string{header}, // This may not be strictly correct, but the result is the same.
			Children: children,
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
