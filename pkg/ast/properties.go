package ast

func (e *Entry) ScheduledFor() (date string, found bool) {
	if e == nil {
		return
	}
	for _, dp := range e.Description {
		if dp.SpecialTag != nil && StringIsScheduled(dp.SpecialTag.Key) {
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
	SliceRemove(&e.Description, func(dp *DescriptionPart) bool {
		return dp.SpecialTag != nil && dp.SpecialTag.Key == key
	})
}

func StringIsScheduled(str string) bool {
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
	for _, b := range g.Blocks {
		for _, e := range b.Children {
			if e != nil {
				length += 1
			}
		}
	}
	return length
}

func SliceRemove[T any](s *[]T, filter func(T) bool) {
	result := make([]T, 0, len(*s)/2)
	for i := range *s {
		if filter((*s)[i]) {
			continue // Removed.
		}
		result = append(result, (*s)[i])
	}
	*s = result
}
