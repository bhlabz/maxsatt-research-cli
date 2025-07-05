package utils

import (
	"sort"
	"time"
)

func SortDates(dates []time.Time, asc bool) []time.Time {
	sort.Slice(dates, func(i, j int) bool {
		if asc {
			return dates[i].Before(dates[j])
		}
		return dates[i].After(dates[j])
	})
	return dates
}

func GetSortedKeys[T any](m map[time.Time]T, asc bool) []time.Time {
	keys := make([]time.Time, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return SortDates(keys, asc)
}
