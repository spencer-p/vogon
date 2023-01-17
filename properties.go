package main

func (e *Entry) ScheduledFor() (date string, found bool) {
	if e == nil {
		return
	}
	for _, dp := range e.Description {
		if dp.SpecialTag != nil && stringIsScheduled(dp.SpecialTag.Key) {
			date = dp.SpecialTag.Value
			found = true
			return
		}
	}
	return
}

func (e *Entry) Tag(key string) (value string, found bool) {
	if e == nil {
		return
	}
	for _, dp := range e.Description {
		if dp.SpecialTag != nil && dp.SpecialTag.Key == key {
			return dp.SpecialTag.Value, true
		}
	}
	return
}

func (e *Entry) RemoveTag(key string) {
	if e == nil {
		return
	}
	sliceRemove(&e.Description, func(dp *DescriptionPart) bool {
		return dp.SpecialTag != nil && dp.SpecialTag.Key == key
	})
}

func stringIsScheduled(str string) bool {
	switch str {
	case "s":
		return true
	case "sched":
		return true
	case "schedule":
		return true
	case "scheduled":
		return true
	default:
		return false
	}
}

func (g *Grouping) Len() int {
	length := 0
	if g == nil {
		return 0
	}
	for _, e := range g.Children {
		if e != nil {
			length += 1
		}
	}
	return length
}
