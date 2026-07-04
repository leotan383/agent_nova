package workflow

import "testing"

func TestGoalMet(t *testing.T) {
	if !GoalMet(DayActivity{ChaptersWritten: 1}, 4000, 1) {
		t.Fatal("chapter goal should met")
	}
	if GoalMet(DayActivity{Words: 100}, 4000, 1) {
		t.Fatal("should not met")
	}
	if !GoalMet(DayActivity{Words: 5000}, 4000, 2) {
		t.Fatal("words goal should met")
	}
}

func TestComputeStreak(t *testing.T) {
	log := ActivityLog{
		Days: map[string]DayActivity{
			"2026-07-02": {ChaptersWritten: 1},
			"2026-07-03": {ChaptersWritten: 1},
			"2026-07-04": {ChaptersWritten: 1},
		},
	}
	cur, longest := ComputeStreak(log, 4000, 1)
	if cur < 2 {
		t.Fatalf("expected streak >= 2, got %d", cur)
	}
	if longest < cur {
		t.Fatalf("longest %d < current %d", longest, cur)
	}
}
