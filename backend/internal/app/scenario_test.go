package app

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func scenarioTestSnapshot() ScenarioSnapshot {
	assignee := "member-1"
	return ScenarioSnapshot{
		Version:   ScenarioSnapshotVersion,
		ProjectID: "project-1",
		Columns: []ScenarioColumn{
			{ID: "column-done", Name: "Done", Position: 1},
			{ID: "column-todo", Name: "Todo", Position: 0},
		},
		Members: []ScenarioMember{{ID: assignee, Name: "Avery"}},
		Tasks: []ScenarioTask{
			{ID: "task-b", Title: "Build", Status: "Todo", EstimateMinutes: 120, AssigneeID: &assignee, DueDate: stringPtr("2026-08-05")},
			{ID: "task-a", Title: "Design", Status: "Done", EstimateMinutes: 60, DueDate: stringPtr("2026-08-01")},
		},
		Dependencies: []ScenarioDependency{{PredecessorTaskID: "task-a", SuccessorTaskID: "task-b"}},
	}
}

func stringPtr(value string) *string { return &value }

func TestScenarioDigestIsIndependentOfInputOrder(t *testing.T) {
	left := scenarioTestSnapshot()
	right := scenarioTestSnapshot()
	left.Tasks[0].Labels = []string{"z", "a"}
	right.Tasks[0].Labels = []string{"a", "z"}
	right.Tasks[0], right.Tasks[1] = right.Tasks[1], right.Tasks[0]
	right.Columns[0], right.Columns[1] = right.Columns[1], right.Columns[0]
	leftDigest, err := digestSnapshot(left)
	if err != nil {
		t.Fatal(err)
	}
	rightDigest, err := digestSnapshot(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftDigest != rightDigest {
		t.Fatalf("digest changed with ordering: %s != %s", leftDigest, rightDigest)
	}
}

func TestScenarioEventsApplyWithoutMutatingTheBase(t *testing.T) {
	state := newScenarioState(scenarioTestSnapshot())
	base := state.Clone()
	events := []ScenarioEventRecord{
		{EventType: "task_delay", Payload: []byte(`{"taskId":"task-b","days":3}`)},
		{EventType: "task_status_change", Payload: []byte(`{"taskId":"task-b","status":"Done"}`)},
		{EventType: "task_estimate_change", Payload: []byte(`{"taskId":"task-b","estimateMinutes":240}`)},
		{EventType: "task_assignee_change", Payload: []byte(`{"taskId":"task-b","assigneeId":null}`)},
	}
	for _, event := range events {
		if err := state.Apply(event); err != nil {
			t.Fatalf("apply %s: %v", event.EventType, err)
		}
	}
	if !reflect.DeepEqual(base.Tasks["task-b"].DueDate, stringPtr("2026-08-05")) {
		t.Fatal("base task changed while applying events")
	}
	if state.Tasks["task-b"].Status != "Done" || state.Tasks["task-b"].DueDate == nil || *state.Tasks["task-b"].DueDate != "2026-08-08" || state.Tasks["task-b"].EstimateMinutes != 240 || state.Tasks["task-b"].AssigneeID != nil {
		t.Fatalf("unexpected derived task: %#v", state.Tasks["task-b"])
	}
}

func TestScenarioAssigneeChangeNormalizesMemberID(t *testing.T) {
	state := newScenarioState(scenarioTestSnapshot())
	mutation, payload, err := decodeScenarioPayload("task_assignee_change", []byte(`{"taskId":"task-b","assigneeId":"  member-1  "}`), state)
	if err != nil {
		t.Fatal(err)
	}
	if mutation.AssigneeID == nil || *mutation.AssigneeID != "member-1" {
		t.Fatalf("mutation assignee = %#v, want trimmed member-1", mutation.AssigneeID)
	}
	var normalized struct {
		TaskID     string  `json:"taskId"`
		AssigneeID *string `json:"assigneeId"`
	}
	if err := json.Unmarshal(payload, &normalized); err != nil {
		t.Fatal(err)
	}
	if normalized.AssigneeID == nil || *normalized.AssigneeID != "member-1" {
		t.Fatalf("normalized payload assignee = %#v, want trimmed member-1", normalized.AssigneeID)
	}
	if err := state.Apply(ScenarioEventRecord{EventType: "task_assignee_change", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	if state.Tasks["task-b"].AssigneeID == nil || *state.Tasks["task-b"].AssigneeID != "member-1" {
		t.Fatalf("replayed assignee = %#v, want trimmed member-1", state.Tasks["task-b"].AssigneeID)
	}
}

func TestScenarioReplayRejectsProjectMismatchAndNonContiguousEvents(t *testing.T) {
	record := scenarioRecord{
		ProjectID: "project-1",
		Snapshot:  scenarioTestSnapshot(),
		Events: []ScenarioEventRecord{
			{Sequence: 1},
			{Sequence: 3},
		},
	}
	if _, err := scenarioStateFromRecord(record); err == nil || !strings.Contains(err.Error(), "not contiguous") {
		t.Fatalf("non-contiguous replay error = %v", err)
	}

	record.Events = nil
	record.Snapshot.ProjectID = "project-2"
	if _, err := scenarioStateFromRecord(record); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("project mismatch replay error = %v", err)
	}
}

func TestScenarioWatermarkResponseIsExplicitlyAdvisory(t *testing.T) {
	response := scenarioResponse(scenarioRecord{ProjectID: "project-1", BaseEventWatermark: 42}, false)
	if _, exists := response["baseEventId"]; exists {
		t.Fatal("response must not expose the misleading reproducibility-boundary field")
	}
	if got, ok := response["baseEventWatermark"].(int64); !ok || got != 42 {
		t.Fatalf("watermark = %#v, want int64(42)", response["baseEventWatermark"])
	}
	if got := response["baseEventWatermarkSemantics"]; got != scenarioBaseWatermarkSemantics {
		t.Fatalf("watermark semantics = %#v, want %q", got, scenarioBaseWatermarkSemantics)
	}
	if got, ok := response["baseEventWatermarkReproducible"].(bool); !ok || got {
		t.Fatalf("watermark reproducibility = %#v, want false", response["baseEventWatermarkReproducible"])
	}
}

func TestScenarioSnapshotUsesRepeatableRead(t *testing.T) {
	options := scenarioSnapshotTxOptions()
	if options.IsoLevel != pgx.RepeatableRead {
		t.Fatalf("snapshot isolation = %q, want %q", options.IsoLevel, pgx.RepeatableRead)
	}
}

func TestCapturedScenarioSnapshotRejectsInvalidDependencies(t *testing.T) {
	tests := []struct {
		name       string
		dependency ScenarioDependency
		want       string
	}{
		{name: "predecessor outside project", dependency: ScenarioDependency{PredecessorTaskID: "foreign-task", SuccessorTaskID: "task-b"}, want: "predecessor"},
		{name: "successor outside project", dependency: ScenarioDependency{PredecessorTaskID: "task-a", SuccessorTaskID: "foreign-task"}, want: "successor"},
		{name: "self reference", dependency: ScenarioDependency{PredecessorTaskID: "task-a", SuccessorTaskID: "task-a"}, want: "self-referential"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := scenarioTestSnapshot()
			snapshot.Dependencies = []ScenarioDependency{test.dependency}
			err := validateCapturedScenarioSnapshot(snapshot)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestScenarioDependencyAddRejectsCycles(t *testing.T) {
	state := newScenarioState(scenarioTestSnapshot())
	err := state.Apply(ScenarioEventRecord{EventType: "dependency_add", Payload: []byte(`{"predecessorTaskId":"task-b","successorTaskId":"task-a"}`)})
	var validation scenarioValidationError
	if !strings.Contains(err.Error(), "cycle") || !errors.As(err, &validation) || !validation.Conflict {
		t.Fatalf("expected cycle conflict, got %v", err)
	}
	if _, exists := state.Dependencies[ScenarioDependency{PredecessorTaskID: "task-b", SuccessorTaskID: "task-a"}]; exists {
		t.Fatal("rejected cycle was applied")
	}
}

func TestAnalyzeScenarioIsDeterministicAndFindsRisks(t *testing.T) {
	snapshot := scenarioTestSnapshot()
	snapshot.Tasks[1].Status = "Todo" // task-a becomes an unfinished prerequisite for task-b.
	snapshot.Tasks = append(snapshot.Tasks,
		ScenarioTask{ID: "task-c", Title: "Ship", Status: "Todo", EstimateMinutes: 2400, AssigneeID: stringPtr("member-1"), DueDate: stringPtr("2026-07-31")},
	)
	state := newScenarioState(snapshot)
	analysis, err := AnalyzeScenario(state, "2026-08-04")
	if err != nil {
		t.Fatal(err)
	}
	if analysis.CriticalPathMinutes != 2400 || !reflect.DeepEqual(analysis.CriticalPath, []string{"task-c"}) {
		t.Fatalf("critical path = %v/%d", analysis.CriticalPath, analysis.CriticalPathMinutes)
	}
	if analysis.Summary.Blocked != 1 || analysis.Summary.Late != 2 || analysis.Summary.OverloadedAssignees != 1 {
		t.Fatalf("summary = %#v", analysis.Summary)
	}
	if len(analysis.AnalysisDigest) != 64 || len(analysis.StateDigest) != 64 {
		t.Fatalf("missing digests: %#v", analysis)
	}
	second, err := AnalyzeScenario(state, "2026-08-04")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(analysis, second) {
		t.Fatal("same state and asOf produced different analysis")
	}
}

func TestCompareScenarioStatesReturnsSortedChanges(t *testing.T) {
	left := newScenarioState(scenarioTestSnapshot())
	right := left.Clone()
	if err := right.Apply(ScenarioEventRecord{EventType: "task_status_change", Payload: []byte(`{"taskId":"task-b","status":"Done"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := right.Apply(ScenarioEventRecord{EventType: "dependency_add", Payload: []byte(`{"predecessorTaskId":"task-b","successorTaskId":"task-a"}`)}); err == nil {
		t.Fatal("expected cycle rejection")
	}
	comparison, err := CompareScenarioStates(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if len(comparison.ChangedTasks) != 1 || comparison.ChangedTasks[0].TaskID != "task-b" || comparison.ChangedTasks[0].After.Status != "Done" {
		t.Fatalf("unexpected comparison: %#v", comparison)
	}
}

func TestScenarioMigrationIsAdditiveAndProtected(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/009_scenario_simulator.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	for _, marker := range []string{
		"add column if not exists estimate_minutes",
		"create table if not exists task_dependencies",
		"create table if not exists scenarios",
		"create table if not exists scenario_events",
		"scenario base snapshot is immutable",
		"scenario events are append-only",
		"not a transactional reproducibility boundary",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("migration missing %q", marker)
		}
	}
}

func TestScenarioPersistenceIntegrityMigrationGuardsDatabaseContracts(t *testing.T) {
	contents, err := os.ReadFile("../../migrations/010_scenario_persistence_integrity.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	for _, marker := range []string{
		"tasks_project_id_id_unique",
		"task_dependencies_predecessor_project_fk",
		"task_dependencies_successor_project_fk",
		"validate constraint task_dependencies_predecessor_project_fk",
		"pg_trigger_depth() > 1",
		"not exists (\n      select 1 from scenarios where id = old.scenario_id",
		"Blocks ordinary scenario event UPDATE/DELETE while allowing FK cascades",
		"not a transactional reproducibility boundary",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("persistence migration missing %q", marker)
		}
	}
}

func TestScenarioRoutesAreProjectScopedAndRegistered(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	defer gin.SetMode(previousMode)
	server := &Server{cfg: Config{FrontendURL: "http://localhost:5173"}}
	server.routes()
	want := map[string]bool{
		"GET /api/projects/:id/scenarios":  false,
		"POST /api/projects/:id/scenarios": false,
		"GET /api/scenarios/:id":           false,
		"POST /api/scenarios/:id/events":   false,
		"GET /api/scenarios/:id/analysis":  false,
		"GET /api/scenarios/:id/compare":   false,
	}
	for _, route := range server.router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("missing route %s", route)
		}
	}
}
