package main

import "testing"

func TestDeadlineOrdering(t *testing.T) {
	got := orderByDeadline([]Course{{ID: "late", Deadline: "2026-10-01"}, {ID: "soon", Deadline: "2026-09-12"}})
	if got[0].ID != "soon" {
		t.Fatalf("expected soon first, got %s", got[0].ID)
	}
}
