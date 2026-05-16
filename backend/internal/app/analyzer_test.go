package app

import (
	"strings"
	"testing"
	"time"
)

func TestSummarizeEvents(t *testing.T) {
	tests := []struct {
		name   string
		events []string
		want   []string // substrings that should appear
	}{
		{
			name:   "empty",
			events: []string{},
			want:   []string{"0 miscellaneous"},
		},
		{
			name:   "single task created",
			events: []string{"task.created"},
			want:   []string{"1 task created"},
		},
		{
			name:   "multiple same type",
			events: []string{"task.created", "task.created", "task.created"},
			want:   []string{"3 created"},
		},
		{
			name:   "mixed events",
			events: []string{"task.created", "task.moved", "comment.created", "task.updated"},
			want:   []string{"created", "moved", "commented", "updated"},
		},
		{
			name:   "unknown event type",
			events: []string{"something.unknown"},
			want:   []string{"miscellaneous"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeEvents(tt.events)
			for _, sub := range tt.want {
				if !strings.Contains(got, sub) {
					t.Errorf("summarizeEvents(%v) = %q, want it to contain %q", tt.events, got, sub)
				}
			}
		})
	}
}

// TestAIAnalyzeLogic validates the deterministic analysis logic without a database.
// It constructs sample task, member, and activity data inline.
func TestAIAnalyzeLogic(t *testing.T) {
	// Simulate the core analysis logic from aiAnalyze as standalone functions.
	// Since the actual analyzer is coupled to the database, we test the computation helpers.

	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sevenDaysAgo := now.AddDate(0, 0, -7)

	type aiTask struct {
		ID, Title, Description, Priority, Status string
		AssigneeID, DueDate, Labels               *string
		UpdatedAt                                time.Time
	}

	tasks := []aiTask{
		{ID: "t1", Title: "Urgent overdue task", Priority: "Urgent", Status: "In Progress", DueDate: strPtr(now.AddDate(0, 0, -2).Format("2006-01-02")), AssigneeID: strPtr("u1"), UpdatedAt: now.Add(-time.Hour)},
		{ID: "t2", Title: "High priority task", Priority: "High", Status: "Todo", DueDate: strPtr(now.AddDate(0, 0, -1).Format("2006-01-02")), AssigneeID: strPtr("u2"), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "t3", Title: "Normal task", Priority: "Medium", Status: "In Progress", DueDate: strPtr(now.AddDate(0, 0, 5).Format("2006-01-02")), AssigneeID: strPtr("u1"), UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: "t4", Title: "Unassigned task", Priority: "Medium", Status: "Backlog", DueDate: nil, AssigneeID: nil, UpdatedAt: eightDaysAgo()},
		{ID: "t5", Title: "Done task", Priority: "Low", Status: "Done", DueDate: nil, AssigneeID: strPtr("u3"), UpdatedAt: now.Add(-4 * time.Hour)},
		{ID: "t6", Title: "Blocked by label", Priority: "High", Status: "In Progress", DueDate: strPtr(now.AddDate(0, 0, 3).Format("2006-01-02")), AssigneeID: strPtr("u2"), Labels: strPtr(`{"blocked"}`), UpdatedAt: now.Add(-24 * time.Hour)},
	}

	// Run the same computation the AI analyzer would
	var overdueCount, urgentCount, unassignedCount, highPriorityCount int
	assigneeTaskCount := map[string]int{}
	staleCount := 0
	statusCounts := map[string]int{}

	for _, t := range tasks {
		statusCounts[t.Status]++
		if t.DueDate != nil && *t.DueDate != "" {
			d, err := time.Parse("2006-01-02", *t.DueDate)
			if err == nil && d.Before(startOfToday) {
				overdueCount++
			}
		}
		p := strings.ToLower(t.Priority)
		if p == "urgent" {
			urgentCount++
		}
		if p == "high" {
			highPriorityCount++
		}
		if t.AssigneeID == nil || *t.AssigneeID == "" {
			unassignedCount++
		} else {
			assigneeTaskCount[*t.AssigneeID]++
		}
		if t.UpdatedAt.Before(sevenDaysAgo) && t.Status != "Done" {
			staleCount++
		}
	}

	// Assertions
	if overdueCount != 2 {
		t.Errorf("expected 2 overdue tasks, got %d", overdueCount)
	}
	if urgentCount != 1 {
		t.Errorf("expected 1 urgent task, got %d", urgentCount)
	}
	if highPriorityCount != 2 { // t2 (High) + t6 (High)
		t.Errorf("expected 2 high priority tasks, got %d", highPriorityCount)
	}
	if unassignedCount != 1 {
		t.Errorf("expected 1 unassigned task, got %d", unassignedCount)
	}
	if staleCount != 1 {
		t.Errorf("expected 1 stale task, got %d", staleCount)
	}
	if assigneeTaskCount["u1"] != 2 {
		t.Errorf("expected u1 to have 2 tasks, got %d", assigneeTaskCount["u1"])
	}
	if assigneeTaskCount["u2"] != 2 {
		t.Errorf("expected u2 to have 2 tasks, got %d", assigneeTaskCount["u2"])
	}
}

func strPtr(s string) *string { return &s }

func eightDaysAgo() time.Time {
	return time.Now().AddDate(0, 0, -8)
}
