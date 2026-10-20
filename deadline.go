package main

import "sort"

func orderByDeadline(in []Course) []Course {
	out := append([]Course(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Deadline < out[j].Deadline })
	return out
}
