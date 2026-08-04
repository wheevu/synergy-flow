package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	ScenarioSnapshotVersion        = 1
	ScenarioCapacityMinutes        = 2400 // 40 hours of open work per assignee.
	scenarioBaseWatermarkSemantics = "advisory"
)

var (
	errScenarioTaskNotFound       = errors.New("scenario task not found")
	errScenarioMemberNotFound     = errors.New("scenario assignee is not a workspace member")
	errScenarioDependencyNotFound = errors.New("scenario dependency not found")
	errScenarioCycle              = errors.New("dependency would create a cycle")
)

type ScenarioColumn struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type ScenarioMember struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ScenarioTask struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        string   `json:"priority"`
	Status          string   `json:"status"`
	AssigneeID      *string  `json:"assigneeId,omitempty"`
	DueDate         *string  `json:"dueDate,omitempty"`
	Labels          []string `json:"labels"`
	Position        int      `json:"position"`
	EstimateMinutes int      `json:"estimateMinutes"`
}

type ScenarioDependency struct {
	PredecessorTaskID string `json:"predecessorTaskId"`
	SuccessorTaskID   string `json:"successorTaskId"`
}

type ScenarioSnapshot struct {
	Version      int                  `json:"version"`
	ProjectID    string               `json:"projectId"`
	Columns      []ScenarioColumn     `json:"columns"`
	Members      []ScenarioMember     `json:"members"`
	Tasks        []ScenarioTask       `json:"tasks"`
	Dependencies []ScenarioDependency `json:"dependencies"`
}

type ScenarioEventRecord struct {
	ID        string
	Sequence  int64
	EventType string
	Payload   json.RawMessage
	CreatedBy string
	CreatedAt time.Time
}

type ScenarioState struct {
	Snapshot     ScenarioSnapshot
	Tasks        map[string]ScenarioTask
	Dependencies map[ScenarioDependency]struct{}
}

type ScenarioSummary struct {
	Total               int `json:"total"`
	Completed           int `json:"completed"`
	Open                int `json:"open"`
	Blocked             int `json:"blocked"`
	Late                int `json:"late"`
	OverloadedAssignees int `json:"overloadedAssignees"`
	CycleNodes          int `json:"cycleNodes"`
}

type ScenarioFinding struct {
	Kind       string   `json:"kind"`
	Severity   string   `json:"severity"`
	Message    string   `json:"message"`
	TaskIDs    []string `json:"taskIds,omitempty"`
	AssigneeID string   `json:"assigneeId,omitempty"`
	Minutes    int      `json:"minutes,omitempty"`
}

type ScenarioAnalysis struct {
	AsOf                string            `json:"asOf"`
	StateDigest         string            `json:"stateDigest"`
	AnalysisDigest      string            `json:"analysisDigest"`
	Summary             ScenarioSummary   `json:"summary"`
	CriticalPath        []string          `json:"criticalPath"`
	CriticalPathMinutes int               `json:"criticalPathMinutes"`
	Findings            []ScenarioFinding `json:"findings"`
}

type ScenarioTaskChange struct {
	TaskID string        `json:"taskId"`
	Title  string        `json:"title"`
	Before *ScenarioTask `json:"before,omitempty"`
	After  *ScenarioTask `json:"after,omitempty"`
}

type ScenarioComparison struct {
	LeftDigest          string               `json:"leftDigest"`
	RightDigest         string               `json:"rightDigest"`
	ChangedTasks        []ScenarioTaskChange `json:"changedTasks"`
	AddedDependencies   []ScenarioDependency `json:"addedDependencies"`
	RemovedDependencies []ScenarioDependency `json:"removedDependencies"`
}

type scenarioMutation struct {
	TaskID            string
	Days              int
	Status            string
	AssigneeID        *string
	EstimateMinutes   int
	PredecessorTaskID string
	SuccessorTaskID   string
}

type scenarioValidationError struct {
	Message      string
	CycleTaskIDs []string
	Conflict     bool
}

func (e scenarioValidationError) Error() string { return e.Message }

func newScenarioState(snapshot ScenarioSnapshot) ScenarioState {
	snapshot = canonicalSnapshot(snapshot)
	tasks := make(map[string]ScenarioTask, len(snapshot.Tasks))
	dependencies := make(map[ScenarioDependency]struct{}, len(snapshot.Dependencies))
	for _, task := range snapshot.Tasks {
		tasks[task.ID] = cloneScenarioTask(task)
	}
	for _, dependency := range snapshot.Dependencies {
		dependencies[dependency] = struct{}{}
	}
	return ScenarioState{Snapshot: snapshot, Tasks: tasks, Dependencies: dependencies}
}

func (s ScenarioState) Clone() ScenarioState {
	clone := newScenarioState(s.Snapshot)
	clone.Tasks = make(map[string]ScenarioTask, len(s.Tasks))
	for id, task := range s.Tasks {
		clone.Tasks[id] = cloneScenarioTask(task)
	}
	clone.Dependencies = make(map[ScenarioDependency]struct{}, len(s.Dependencies))
	for dependency := range s.Dependencies {
		clone.Dependencies[dependency] = struct{}{}
	}
	return clone
}

func cloneScenarioTask(task ScenarioTask) ScenarioTask {
	copy := task
	copy.Labels = append([]string(nil), task.Labels...)
	if task.AssigneeID != nil {
		value := *task.AssigneeID
		copy.AssigneeID = &value
	}
	if task.DueDate != nil {
		value := *task.DueDate
		copy.DueDate = &value
	}
	return copy
}

func canonicalSnapshot(snapshot ScenarioSnapshot) ScenarioSnapshot {
	if snapshot.Version == 0 {
		snapshot.Version = ScenarioSnapshotVersion
	}
	snapshot.Columns = append([]ScenarioColumn(nil), snapshot.Columns...)
	snapshot.Members = append([]ScenarioMember(nil), snapshot.Members...)
	snapshot.Tasks = append([]ScenarioTask(nil), snapshot.Tasks...)
	snapshot.Dependencies = append([]ScenarioDependency(nil), snapshot.Dependencies...)
	sort.Slice(snapshot.Columns, func(i, j int) bool {
		if snapshot.Columns[i].Position != snapshot.Columns[j].Position {
			return snapshot.Columns[i].Position < snapshot.Columns[j].Position
		}
		return snapshot.Columns[i].ID < snapshot.Columns[j].ID
	})
	sort.Slice(snapshot.Members, func(i, j int) bool { return snapshot.Members[i].ID < snapshot.Members[j].ID })
	sort.Slice(snapshot.Tasks, func(i, j int) bool { return snapshot.Tasks[i].ID < snapshot.Tasks[j].ID })
	sort.Slice(snapshot.Dependencies, func(i, j int) bool {
		if snapshot.Dependencies[i].PredecessorTaskID != snapshot.Dependencies[j].PredecessorTaskID {
			return snapshot.Dependencies[i].PredecessorTaskID < snapshot.Dependencies[j].PredecessorTaskID
		}
		return snapshot.Dependencies[i].SuccessorTaskID < snapshot.Dependencies[j].SuccessorTaskID
	})
	for i := range snapshot.Tasks {
		snapshot.Tasks[i] = cloneScenarioTask(snapshot.Tasks[i])
		sort.Strings(snapshot.Tasks[i].Labels)
	}
	return snapshot
}

func (s ScenarioState) SnapshotValue() ScenarioSnapshot {
	tasks := make([]ScenarioTask, 0, len(s.Tasks))
	for _, task := range s.Tasks {
		tasks = append(tasks, cloneScenarioTask(task))
	}
	dependencies := make([]ScenarioDependency, 0, len(s.Dependencies))
	for dependency := range s.Dependencies {
		dependencies = append(dependencies, dependency)
	}
	return canonicalSnapshot(ScenarioSnapshot{
		Version:      ScenarioSnapshotVersion,
		ProjectID:    s.Snapshot.ProjectID,
		Columns:      s.Snapshot.Columns,
		Members:      s.Snapshot.Members,
		Tasks:        tasks,
		Dependencies: dependencies,
	})
}

func canonicalBytes(value any) ([]byte, error) { return json.Marshal(value) }

func digestValue(value any) (string, error) {
	encoded, err := canonicalBytes(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func digestSnapshot(snapshot ScenarioSnapshot) (string, error) {
	return digestValue(canonicalSnapshot(snapshot))
}

func (s ScenarioState) Digest() (string, error) { return digestValue(s.SnapshotValue()) }

func decodeScenarioPayload(eventType string, payload []byte, state ScenarioState) (scenarioMutation, []byte, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return scenarioMutation{}, nil, scenarioValidationError{Message: "event payload is required"}
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	mutation := scenarioMutation{}
	switch eventType {
	case "task_delay":
		var input struct {
			TaskID string `json:"taskId"`
			Days   int    `json:"days"`
		}
		if err := decoder.Decode(&input); err != nil {
			return mutation, nil, scenarioValidationError{Message: "invalid task_delay payload"}
		}
		mutation.TaskID, mutation.Days = strings.TrimSpace(input.TaskID), input.Days
		if mutation.Days == 0 || mutation.Days < -3650 || mutation.Days > 3650 {
			return mutation, nil, scenarioValidationError{Message: "task delay days must be between -3650 and 3650 and non-zero"}
		}
		encoded, _ := json.Marshal(input)
		return validateScenarioTaskMutation(state, mutation, encoded)
	case "task_status_change":
		var input struct {
			TaskID string `json:"taskId"`
			Status string `json:"status"`
		}
		if err := decoder.Decode(&input); err != nil {
			return mutation, nil, scenarioValidationError{Message: "invalid task_status_change payload"}
		}
		mutation.TaskID, mutation.Status = strings.TrimSpace(input.TaskID), strings.TrimSpace(input.Status)
		if mutation.Status == "" {
			return mutation, nil, scenarioValidationError{Message: "task status is required"}
		}
		for _, column := range state.Snapshot.Columns {
			if strings.EqualFold(column.Name, mutation.Status) {
				mutation.Status = column.Name
				break
			}
		}
		if !scenarioStatusExists(state.Snapshot.Columns, mutation.Status) {
			return mutation, nil, scenarioValidationError{Message: "task status is not a column in this project"}
		}
		encoded, _ := json.Marshal(struct {
			TaskID string `json:"taskId"`
			Status string `json:"status"`
		}{mutation.TaskID, mutation.Status})
		return validateScenarioTaskMutation(state, mutation, encoded)
	case "task_assignee_change":
		var input struct {
			TaskID     string  `json:"taskId"`
			AssigneeID *string `json:"assigneeId"`
		}
		if err := decoder.Decode(&input); err != nil {
			return mutation, nil, scenarioValidationError{Message: "invalid task_assignee_change payload"}
		}
		mutation.TaskID = strings.TrimSpace(input.TaskID)
		if input.AssigneeID != nil {
			value := strings.TrimSpace(*input.AssigneeID)
			if value == "" {
				input.AssigneeID = nil
			} else if !scenarioMemberExists(state.Snapshot.Members, value) {
				return mutation, nil, errScenarioMemberNotFound
			} else {
				// Store and replay the canonical member ID, not the caller's
				// whitespace-padded representation.
				input.AssigneeID = &value
			}
		}
		mutation.AssigneeID = input.AssigneeID
		encoded, _ := json.Marshal(input)
		return validateScenarioTaskMutation(state, mutation, encoded)
	case "task_estimate_change":
		var input struct {
			TaskID          string `json:"taskId"`
			EstimateMinutes int    `json:"estimateMinutes"`
		}
		if err := decoder.Decode(&input); err != nil {
			return mutation, nil, scenarioValidationError{Message: "invalid task_estimate_change payload"}
		}
		mutation.TaskID, mutation.EstimateMinutes = strings.TrimSpace(input.TaskID), input.EstimateMinutes
		if mutation.EstimateMinutes < 0 || mutation.EstimateMinutes > 10080 {
			return mutation, nil, scenarioValidationError{Message: "estimateMinutes must be between 0 and 10080"}
		}
		encoded, _ := json.Marshal(input)
		return validateScenarioTaskMutation(state, mutation, encoded)
	case "dependency_add", "dependency_remove":
		var input struct {
			PredecessorTaskID string `json:"predecessorTaskId"`
			SuccessorTaskID   string `json:"successorTaskId"`
		}
		if err := decoder.Decode(&input); err != nil {
			return mutation, nil, scenarioValidationError{Message: "invalid dependency payload"}
		}
		mutation.PredecessorTaskID = strings.TrimSpace(input.PredecessorTaskID)
		mutation.SuccessorTaskID = strings.TrimSpace(input.SuccessorTaskID)
		if mutation.PredecessorTaskID == "" || mutation.SuccessorTaskID == "" || mutation.PredecessorTaskID == mutation.SuccessorTaskID {
			return mutation, nil, scenarioValidationError{Message: "dependency endpoints must be two different tasks"}
		}
		if _, ok := state.Tasks[mutation.PredecessorTaskID]; !ok {
			return mutation, nil, errScenarioTaskNotFound
		}
		if _, ok := state.Tasks[mutation.SuccessorTaskID]; !ok {
			return mutation, nil, errScenarioTaskNotFound
		}
		dependency := ScenarioDependency{PredecessorTaskID: mutation.PredecessorTaskID, SuccessorTaskID: mutation.SuccessorTaskID}
		_, exists := state.Dependencies[dependency]
		if eventType == "dependency_add" && exists {
			return mutation, nil, scenarioValidationError{Message: "dependency already exists"}
		}
		if eventType == "dependency_remove" && !exists {
			return mutation, nil, errScenarioDependencyNotFound
		}
		encoded, _ := json.Marshal(input)
		return mutation, encoded, nil
	default:
		return mutation, nil, scenarioValidationError{Message: "unsupported scenario event type"}
	}
}

func validateScenarioTaskMutation(state ScenarioState, mutation scenarioMutation, encoded []byte) (scenarioMutation, []byte, error) {
	if mutation.TaskID == "" {
		return mutation, nil, scenarioValidationError{Message: "taskId is required"}
	}
	if _, ok := state.Tasks[mutation.TaskID]; !ok {
		return mutation, nil, errScenarioTaskNotFound
	}
	return mutation, encoded, nil
}

func scenarioStatusExists(columns []ScenarioColumn, status string) bool {
	for _, column := range columns {
		if column.Name == status {
			return true
		}
	}
	return false
}

func scenarioMemberExists(members []ScenarioMember, id string) bool {
	for _, member := range members {
		if member.ID == id {
			return true
		}
	}
	return false
}

func (s *ScenarioState) Apply(event ScenarioEventRecord) error {
	mutation, _, err := decodeScenarioPayload(event.EventType, event.Payload, *s)
	if err != nil {
		return err
	}
	candidate := s.Clone()
	switch event.EventType {
	case "task_delay":
		task := candidate.Tasks[mutation.TaskID]
		if task.DueDate != nil {
			date, parseErr := time.Parse("2006-01-02", *task.DueDate)
			if parseErr != nil {
				return scenarioValidationError{Message: "task due date is invalid in the base snapshot"}
			}
			shifted := date.AddDate(0, 0, mutation.Days).Format("2006-01-02")
			task.DueDate = &shifted
		}
		candidate.Tasks[mutation.TaskID] = task
	case "task_status_change":
		task := candidate.Tasks[mutation.TaskID]
		task.Status = mutation.Status
		candidate.Tasks[mutation.TaskID] = task
	case "task_assignee_change":
		task := candidate.Tasks[mutation.TaskID]
		task.AssigneeID = mutation.AssigneeID
		candidate.Tasks[mutation.TaskID] = task
	case "task_estimate_change":
		task := candidate.Tasks[mutation.TaskID]
		task.EstimateMinutes = mutation.EstimateMinutes
		candidate.Tasks[mutation.TaskID] = task
	case "dependency_add":
		dependency := ScenarioDependency{PredecessorTaskID: mutation.PredecessorTaskID, SuccessorTaskID: mutation.SuccessorTaskID}
		candidate.Dependencies[dependency] = struct{}{}
		if cycle := candidate.CycleNodes(); len(cycle) > 0 {
			return scenarioValidationError{Message: errScenarioCycle.Error(), CycleTaskIDs: cycle, Conflict: true}
		}
	case "dependency_remove":
		delete(candidate.Dependencies, ScenarioDependency{PredecessorTaskID: mutation.PredecessorTaskID, SuccessorTaskID: mutation.SuccessorTaskID})
	}
	*s = candidate
	return nil
}

func (s ScenarioState) CycleNodes() []string {
	indegree := make(map[string]int, len(s.Tasks))
	adjacency := make(map[string][]string, len(s.Tasks))
	for id := range s.Tasks {
		indegree[id] = 0
	}
	for dependency := range s.Dependencies {
		if _, predOK := s.Tasks[dependency.PredecessorTaskID]; !predOK {
			continue
		}
		if _, succOK := s.Tasks[dependency.SuccessorTaskID]; !succOK {
			continue
		}
		adjacency[dependency.PredecessorTaskID] = append(adjacency[dependency.PredecessorTaskID], dependency.SuccessorTaskID)
		indegree[dependency.SuccessorTaskID]++
	}
	queue := make([]string, 0)
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	processed := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		processed++
		neighbors := append([]string(nil), adjacency[id]...)
		sort.Strings(neighbors)
		for _, next := range neighbors {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
				sort.Strings(queue)
			}
		}
	}
	if processed == len(indegree) {
		return nil
	}
	cycle := make([]string, 0, len(indegree)-processed)
	for id, degree := range indegree {
		if degree > 0 {
			cycle = append(cycle, id)
		}
	}
	sort.Strings(cycle)
	return cycle
}

func scenarioDone(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "done") || strings.EqualFold(strings.TrimSpace(status), "completed")
}

func scenarioPredecessors(state ScenarioState) map[string][]string {
	predecessors := make(map[string][]string, len(state.Tasks))
	for dependency := range state.Dependencies {
		predecessors[dependency.SuccessorTaskID] = append(predecessors[dependency.SuccessorTaskID], dependency.PredecessorTaskID)
	}
	for taskID := range predecessors {
		sort.Strings(predecessors[taskID])
	}
	return predecessors
}

func (s ScenarioState) CriticalPath() ([]string, int) {
	if len(s.CycleNodes()) > 0 || len(s.Tasks) == 0 {
		return nil, 0
	}
	indegree := make(map[string]int, len(s.Tasks))
	adjacency := make(map[string][]string, len(s.Tasks))
	for id := range s.Tasks {
		indegree[id] = 0
	}
	for dependency := range s.Dependencies {
		if _, ok := s.Tasks[dependency.PredecessorTaskID]; !ok {
			continue
		}
		if _, ok := s.Tasks[dependency.SuccessorTaskID]; !ok {
			continue
		}
		indegree[dependency.SuccessorTaskID]++
		adjacency[dependency.PredecessorTaskID] = append(adjacency[dependency.PredecessorTaskID], dependency.SuccessorTaskID)
	}
	queue := make([]string, 0)
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	finish := make(map[string]int, len(s.Tasks))
	paths := make(map[string][]string, len(s.Tasks))
	for _, task := range s.Tasks {
		finish[task.ID] = task.EstimateMinutes
		paths[task.ID] = []string{task.ID}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		neighbors := append([]string(nil), adjacency[id]...)
		sort.Strings(neighbors)
		for _, next := range neighbors {
			task := s.Tasks[next]
			candidateFinish := finish[id] + task.EstimateMinutes
			candidatePath := append(append([]string(nil), paths[id]...), next)
			if candidateFinish > finish[next] || (candidateFinish == finish[next] && pathLess(candidatePath, paths[next])) {
				finish[next] = candidateFinish
				paths[next] = candidatePath
			}
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
				sort.Strings(queue)
			}
		}
	}
	var best []string
	bestMinutes := 0
	for id, minutes := range finish {
		if minutes > bestMinutes || (minutes == bestMinutes && pathLess(paths[id], best)) {
			bestMinutes = minutes
			best = append([]string(nil), paths[id]...)
		}
	}
	return best, bestMinutes
}

func pathLess(left, right []string) bool {
	if len(right) == 0 {
		return true
	}
	for i := 0; i < len(left) && i < len(right); i++ {
		if left[i] != right[i] {
			return left[i] < right[i]
		}
	}
	return len(left) < len(right)
}

func AnalyzeScenario(state ScenarioState, asOf string) (ScenarioAnalysis, error) {
	if asOf == "" {
		asOf = time.Now().UTC().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", asOf); err != nil {
		return ScenarioAnalysis{}, fmt.Errorf("invalid asOf date: %w", err)
	}
	state = state.Clone()
	stateDigest, err := state.Digest()
	if err != nil {
		return ScenarioAnalysis{}, err
	}
	predecessors := scenarioPredecessors(state)
	summary := ScenarioSummary{Total: len(state.Tasks)}
	blockedIDs := make([]string, 0)
	lateIDs := make([]string, 0)
	workload := make(map[string]int)
	for id, task := range state.Tasks {
		if scenarioDone(task.Status) {
			summary.Completed++
			continue
		}
		summary.Open++
		blocked := false
		for _, predecessorID := range predecessors[id] {
			if predecessor, ok := state.Tasks[predecessorID]; ok && !scenarioDone(predecessor.Status) {
				blocked = true
				break
			}
		}
		if blocked {
			blockedIDs = append(blockedIDs, id)
		}
		if task.DueDate != nil {
			if dueDate, parseErr := time.Parse("2006-01-02", *task.DueDate); parseErr == nil {
				asOfDate, _ := time.Parse("2006-01-02", asOf)
				if dueDate.Before(asOfDate) {
					lateIDs = append(lateIDs, id)
				}
			}
		}
		if task.AssigneeID != nil && *task.AssigneeID != "" {
			workload[*task.AssigneeID] += task.EstimateMinutes
		}
	}
	sort.Strings(blockedIDs)
	sort.Strings(lateIDs)
	cycleNodes := state.CycleNodes()
	summary.Blocked = len(blockedIDs)
	summary.Late = len(lateIDs)
	summary.CycleNodes = len(cycleNodes)

	findings := make([]ScenarioFinding, 0, len(blockedIDs)+len(lateIDs)+len(workload)+2)
	if len(cycleNodes) > 0 {
		findings = append(findings, ScenarioFinding{Kind: "cycle", Severity: "high", Message: "Dependency graph contains a cycle.", TaskIDs: cycleNodes})
	}
	criticalPath, criticalMinutes := state.CriticalPath()
	if len(criticalPath) > 0 {
		findings = append(findings, ScenarioFinding{Kind: "critical_path", Severity: "info", Message: "Longest dependency path by estimated minutes.", TaskIDs: criticalPath, Minutes: criticalMinutes})
	}
	for _, id := range blockedIDs {
		findings = append(findings, ScenarioFinding{Kind: "blocked", Severity: "high", Message: "Waiting on an unfinished dependency.", TaskIDs: []string{id}})
	}
	for _, id := range lateIDs {
		findings = append(findings, ScenarioFinding{Kind: "late", Severity: "medium", Message: "Due date is before the analysis date.", TaskIDs: []string{id}})
	}
	overloaded := make([]string, 0)
	for assigneeID, minutes := range workload {
		if minutes > ScenarioCapacityMinutes {
			overloaded = append(overloaded, assigneeID)
		}
	}
	sort.Strings(overloaded)
	summary.OverloadedAssignees = len(overloaded)
	for _, assigneeID := range overloaded {
		findings = append(findings, ScenarioFinding{Kind: "overload", Severity: "medium", Message: "Open estimated work exceeds the 40-hour capacity window.", AssigneeID: assigneeID, Minutes: workload[assigneeID]})
	}
	analysis := ScenarioAnalysis{AsOf: asOf, StateDigest: stateDigest, Summary: summary, CriticalPath: criticalPath, CriticalPathMinutes: criticalMinutes, Findings: findings}
	analysisDigest, err := digestValue(analysis)
	if err != nil {
		return ScenarioAnalysis{}, err
	}
	analysis.AnalysisDigest = analysisDigest
	return analysis, nil
}

func CompareScenarioStates(left, right ScenarioState) (ScenarioComparison, error) {
	leftDigest, err := left.Digest()
	if err != nil {
		return ScenarioComparison{}, err
	}
	rightDigest, err := right.Digest()
	if err != nil {
		return ScenarioComparison{}, err
	}
	changes := make([]ScenarioTaskChange, 0)
	ids := make(map[string]struct{}, len(left.Tasks)+len(right.Tasks))
	for id := range left.Tasks {
		ids[id] = struct{}{}
	}
	for id := range right.Tasks {
		ids[id] = struct{}{}
	}
	sortedIDs := make([]string, 0, len(ids))
	for id := range ids {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)
	for _, id := range sortedIDs {
		before, beforeOK := left.Tasks[id]
		after, afterOK := right.Tasks[id]
		if beforeOK && afterOK && reflect.DeepEqual(before, after) || !beforeOK && !afterOK {
			continue
		}
		change := ScenarioTaskChange{TaskID: id}
		if beforeOK {
			copy := cloneScenarioTask(before)
			change.Before = &copy
			change.Title = before.Title
		}
		if afterOK {
			copy := cloneScenarioTask(after)
			change.After = &copy
			if change.Title == "" {
				change.Title = after.Title
			}
		}
		changes = append(changes, change)
	}
	added, removed := compareDependencies(left.Dependencies, right.Dependencies)
	return ScenarioComparison{LeftDigest: leftDigest, RightDigest: rightDigest, ChangedTasks: changes, AddedDependencies: added, RemovedDependencies: removed}, nil
}

func compareDependencies(left, right map[ScenarioDependency]struct{}) ([]ScenarioDependency, []ScenarioDependency) {
	added := make([]ScenarioDependency, 0)
	removed := make([]ScenarioDependency, 0)
	for dependency := range right {
		if _, exists := left[dependency]; !exists {
			added = append(added, dependency)
		}
	}
	for dependency := range left {
		if _, exists := right[dependency]; !exists {
			removed = append(removed, dependency)
		}
	}
	sort.Slice(added, func(i, j int) bool { return dependencyLess(added[i], added[j]) })
	sort.Slice(removed, func(i, j int) bool { return dependencyLess(removed[i], removed[j]) })
	return added, removed
}

func dependencyLess(left, right ScenarioDependency) bool {
	if left.PredecessorTaskID != right.PredecessorTaskID {
		return left.PredecessorTaskID < right.PredecessorTaskID
	}
	return left.SuccessorTaskID < right.SuccessorTaskID
}

type scenarioRecord struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	CreatedBy   string
	// BaseEventWatermark is read from the separate filesystem event log while
	// the PostgreSQL snapshot is being captured. It is diagnostic only; it is
	// not a transactional replay boundary.
	BaseEventWatermark int64
	BaseDigest         string
	Snapshot           ScenarioSnapshot
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Events             []ScenarioEventRecord
}

type scenarioEventResponse struct {
	ID        string          `json:"id"`
	Sequence  int64           `json:"sequence"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedBy string          `json:"createdBy"`
	CreatedAt time.Time       `json:"createdAt"`
}

func scenarioEventResponseFromRecord(event ScenarioEventRecord) scenarioEventResponse {
	return scenarioEventResponse{ID: event.ID, Sequence: event.Sequence, Type: event.EventType, Payload: event.Payload, CreatedBy: event.CreatedBy, CreatedAt: event.CreatedAt}
}

func scenarioResponse(record scenarioRecord, includeEvents bool) gin.H {
	response := gin.H{
		"id":          record.ID,
		"projectId":   record.ProjectID,
		"name":        record.Name,
		"description": record.Description,
		"createdBy":   record.CreatedBy,
		"baseDigest":  record.BaseDigest,
		"createdAt":   record.CreatedAt,
		"updatedAt":   record.UpdatedAt,
		"eventCount":  len(record.Events),
	}
	for key, value := range scenarioBaseWatermarkResponse(record.BaseEventWatermark) {
		response[key] = value
	}
	if includeEvents {
		events := make([]scenarioEventResponse, 0, len(record.Events))
		for _, event := range record.Events {
			events = append(events, scenarioEventResponseFromRecord(event))
		}
		response["events"] = events
	}
	return response
}

// scenarioBaseWatermarkResponse deliberately does not expose the old
// baseEventId name. That value came from a filesystem log which cannot commit
// atomically with the PostgreSQL snapshot, so callers must not mistake it for
// a reproducibility guarantee.
func scenarioBaseWatermarkResponse(watermark int64) gin.H {
	return gin.H{
		"baseEventWatermark":             watermark,
		"baseEventWatermarkSemantics":    scenarioBaseWatermarkSemantics,
		"baseEventWatermarkReproducible": false,
	}
}

func scenarioSnapshotTxOptions() pgx.TxOptions {
	// A scenario is assembled by several queries. REPEATABLE READ guarantees
	// that every query observes the same PostgreSQL snapshot while still
	// allowing the immutable scenario row to be inserted in this transaction.
	return pgx.TxOptions{IsoLevel: pgx.RepeatableRead}
}

func validateCapturedScenarioSnapshot(snapshot ScenarioSnapshot) error {
	if strings.TrimSpace(snapshot.ProjectID) == "" {
		return errors.New("captured scenario project id is required")
	}

	taskIDs := make(map[string]struct{}, len(snapshot.Tasks))
	for _, task := range snapshot.Tasks {
		if strings.TrimSpace(task.ID) == "" {
			return errors.New("captured scenario contains a task without an id")
		}
		if _, exists := taskIDs[task.ID]; exists {
			return fmt.Errorf("captured scenario contains duplicate task %q", task.ID)
		}
		taskIDs[task.ID] = struct{}{}
	}

	dependencies := make(map[ScenarioDependency]struct{}, len(snapshot.Dependencies))
	for _, dependency := range snapshot.Dependencies {
		if dependency.PredecessorTaskID == dependency.SuccessorTaskID {
			return fmt.Errorf("captured scenario dependency for task %q is self-referential", dependency.PredecessorTaskID)
		}
		if _, exists := taskIDs[dependency.PredecessorTaskID]; !exists {
			return fmt.Errorf("captured scenario dependency predecessor %q is outside project %q", dependency.PredecessorTaskID, snapshot.ProjectID)
		}
		if _, exists := taskIDs[dependency.SuccessorTaskID]; !exists {
			return fmt.Errorf("captured scenario dependency successor %q is outside project %q", dependency.SuccessorTaskID, snapshot.ProjectID)
		}
		if _, exists := dependencies[dependency]; exists {
			return fmt.Errorf("captured scenario contains duplicate dependency %q -> %q", dependency.PredecessorTaskID, dependency.SuccessorTaskID)
		}
		dependencies[dependency] = struct{}{}
	}

	return nil
}

func (s *Server) captureScenarioSnapshot(ctx context.Context, tx pgx.Tx, projectID string) (ScenarioSnapshot, error) {
	snapshot := ScenarioSnapshot{Version: ScenarioSnapshotVersion, ProjectID: projectID}
	var workspaceID string
	if err := tx.QueryRow(ctx, "select workspace_id::text from projects where id=$1", projectID).Scan(&workspaceID); err != nil {
		return snapshot, err
	}
	columnRows, err := tx.Query(ctx, "select id::text,name,position from project_columns where project_id=$1 order by position,id", projectID)
	if err != nil {
		return snapshot, err
	}
	for columnRows.Next() {
		var column ScenarioColumn
		if err := columnRows.Scan(&column.ID, &column.Name, &column.Position); err != nil {
			columnRows.Close()
			return snapshot, err
		}
		snapshot.Columns = append(snapshot.Columns, column)
	}
	if err := columnRows.Err(); err != nil {
		columnRows.Close()
		return snapshot, err
	}
	columnRows.Close()

	taskRows, err := tx.Query(ctx, `select t.id::text,t.title,t.description,t.priority,coalesce(t.assignee_id::text,''),coalesce(t.due_date::date::text,''),t.labels,t.position,t.estimate_minutes,c.name from tasks t join project_columns c on c.id=t.column_id where t.project_id=$1 order by t.id`, projectID)
	if err != nil {
		return snapshot, err
	}
	for taskRows.Next() {
		var task ScenarioTask
		var assigneeID, dueDate string
		if err := taskRows.Scan(&task.ID, &task.Title, &task.Description, &task.Priority, &assigneeID, &dueDate, &task.Labels, &task.Position, &task.EstimateMinutes, &task.Status); err != nil {
			taskRows.Close()
			return snapshot, err
		}
		if assigneeID != "" {
			task.AssigneeID = &assigneeID
		}
		if dueDate != "" {
			task.DueDate = &dueDate
		}
		snapshot.Tasks = append(snapshot.Tasks, task)
	}
	if err := taskRows.Err(); err != nil {
		taskRows.Close()
		return snapshot, err
	}
	taskRows.Close()

	memberRows, err := tx.Query(ctx, "select u.id::text,u.name from workspace_members wm join users u on u.id=wm.user_id where wm.workspace_id=$1 order by u.id", workspaceID)
	if err != nil {
		return snapshot, err
	}
	for memberRows.Next() {
		var member ScenarioMember
		if err := memberRows.Scan(&member.ID, &member.Name); err != nil {
			memberRows.Close()
			return snapshot, err
		}
		snapshot.Members = append(snapshot.Members, member)
	}
	if err := memberRows.Err(); err != nil {
		memberRows.Close()
		return snapshot, err
	}
	memberRows.Close()

	dependencyRows, err := tx.Query(ctx, "select predecessor_task_id::text,successor_task_id::text from task_dependencies where project_id=$1 order by predecessor_task_id,successor_task_id", projectID)
	if err != nil {
		return snapshot, err
	}
	for dependencyRows.Next() {
		var dependency ScenarioDependency
		if err := dependencyRows.Scan(&dependency.PredecessorTaskID, &dependency.SuccessorTaskID); err != nil {
			dependencyRows.Close()
			return snapshot, err
		}
		snapshot.Dependencies = append(snapshot.Dependencies, dependency)
	}
	if err := dependencyRows.Err(); err != nil {
		dependencyRows.Close()
		return snapshot, err
	}
	dependencyRows.Close()
	snapshot = canonicalSnapshot(snapshot)
	if err := validateCapturedScenarioSnapshot(snapshot); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (s *Server) loadScenario(ctx context.Context, scenarioID string) (scenarioRecord, error) {
	var record scenarioRecord
	var snapshotBytes []byte
	err := s.db.QueryRow(ctx, `select id::text,project_id::text,name,description,created_by::text,base_event_id,base_snapshot,base_digest,created_at,updated_at from scenarios where id=$1`, scenarioID).Scan(&record.ID, &record.ProjectID, &record.Name, &record.Description, &record.CreatedBy, &record.BaseEventWatermark, &snapshotBytes, &record.BaseDigest, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(snapshotBytes, &record.Snapshot); err != nil {
		return record, fmt.Errorf("decode scenario snapshot: %w", err)
	}
	if err := validateScenarioRecord(record); err != nil {
		return record, err
	}
	if digest, digestErr := digestSnapshot(record.Snapshot); digestErr != nil {
		return record, digestErr
	} else if digest != record.BaseDigest {
		return record, errors.New("scenario base snapshot digest mismatch")
	}
	rows, err := s.db.Query(ctx, `select id::text,sequence,event_type,payload,created_by::text,created_at from scenario_events where scenario_id=$1 order by sequence`, scenarioID)
	if err != nil {
		return record, err
	}
	defer rows.Close()
	for rows.Next() {
		var event ScenarioEventRecord
		if err := rows.Scan(&event.ID, &event.Sequence, &event.EventType, &event.Payload, &event.CreatedBy, &event.CreatedAt); err != nil {
			return record, err
		}
		record.Events = append(record.Events, event)
	}
	if err := rows.Err(); err != nil {
		return record, err
	}
	if err := validateScenarioRecord(record); err != nil {
		return record, err
	}
	return record, nil
}

func scenarioStateFromRecord(record scenarioRecord) (ScenarioState, error) {
	if err := validateScenarioRecord(record); err != nil {
		return ScenarioState{}, err
	}
	state := newScenarioState(record.Snapshot)
	for _, event := range record.Events {
		if err := state.Apply(event); err != nil {
			return state, fmt.Errorf("apply scenario event %d: %w", event.Sequence, err)
		}
	}
	return state, nil
}

func validateScenarioRecord(record scenarioRecord) error {
	if strings.TrimSpace(record.ProjectID) == "" {
		return errors.New("scenario project id is required")
	}
	if strings.TrimSpace(record.Snapshot.ProjectID) != record.ProjectID {
		return fmt.Errorf("scenario snapshot project id %q does not match scenario project id %q", record.Snapshot.ProjectID, record.ProjectID)
	}
	for index, event := range record.Events {
		expected := int64(index + 1)
		if event.Sequence != expected {
			return fmt.Errorf("scenario event sequence is not contiguous: expected %d, got %d", expected, event.Sequence)
		}
	}
	return nil
}

func (s *Server) scenarioProjectID(c *gin.Context, scenarioID string) (string, bool) {
	var projectID string
	if err := s.db.QueryRow(c, "select project_id::text from scenarios where id=$1", scenarioID).Scan(&projectID); err != nil {
		fail(c, 404, err)
		return "", false
	}
	return projectID, true
}

func (s *Server) canScenario(c *gin.Context, scenarioID, need string) (string, bool) {
	projectID, ok := s.scenarioProjectID(c, scenarioID)
	if !ok || !s.canProject(c, projectID, need) {
		return "", false
	}
	return projectID, true
}

func (s *Server) listScenarios(c *gin.Context) {
	projectID := c.Param("id")
	if !s.canProject(c, projectID, "Viewer") {
		return
	}
	rows, err := s.db.Query(c, `select s.id::text,s.project_id::text,s.name,s.description,s.created_by::text,s.base_event_id,s.base_digest,s.created_at,s.updated_at,count(se.id) from scenarios s left join scenario_events se on se.scenario_id=s.id where s.project_id=$1 group by s.id order by s.updated_at desc,s.id`, projectID)
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer rows.Close()
	out := make([]gin.H, 0)
	for rows.Next() {
		var record scenarioRecord
		var eventCount int64
		if err := rows.Scan(&record.ID, &record.ProjectID, &record.Name, &record.Description, &record.CreatedBy, &record.BaseEventWatermark, &record.BaseDigest, &record.CreatedAt, &record.UpdatedAt, &eventCount); err != nil {
			fail(c, 500, err)
			return
		}
		item := scenarioResponse(record, false)
		item["eventCount"] = eventCount
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(200, out)
}

func (s *Server) createScenario(c *gin.Context) {
	projectID := c.Param("id")
	if !s.canProject(c, projectID, "Member") {
		return
	}
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if bind(c, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(400, gin.H{"error": "scenario name is required"})
		return
	}
	tx, err := s.db.BeginTx(c, scenarioSnapshotTxOptions())
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer tx.Rollback(c)
	snapshot, err := s.captureScenarioSnapshot(c, tx, projectID)
	if err != nil {
		fail(c, 500, err)
		return
	}
	baseEventWatermark := int64(0)
	if s.eventLog != nil {
		watermark, watermarkErr := s.eventLog.LastID(projectID)
		if watermarkErr != nil {
			fail(c, 500, watermarkErr)
			return
		}
		if watermark > uint64(1<<63-1) {
			fail(c, 500, errors.New("event log watermark exceeds scenario storage range"))
			return
		}
		baseEventWatermark = int64(watermark)
	}
	baseDigest, err := digestSnapshot(snapshot)
	if err != nil {
		fail(c, 500, err)
		return
	}
	snapshotBytes, err := canonicalBytes(snapshot)
	if err != nil {
		fail(c, 500, err)
		return
	}
	var scenarioID string
	var createdAt, updatedAt time.Time
	err = tx.QueryRow(c, `insert into scenarios(project_id,name,description,created_by,base_event_id,base_snapshot,base_digest) values($1,$2,$3,$4,$5,$6,$7) returning id::text,created_at,updated_at`, projectID, input.Name, input.Description, userID(c), baseEventWatermark, snapshotBytes, baseDigest).Scan(&scenarioID, &createdAt, &updatedAt)
	if err != nil {
		fail(c, 400, err)
		return
	}
	if err := tx.Commit(c); err != nil {
		fail(c, 500, err)
		return
	}
	response := gin.H{"id": scenarioID, "projectId": projectID, "name": input.Name, "description": input.Description, "createdBy": userID(c), "baseDigest": baseDigest, "eventCount": 0, "createdAt": createdAt, "updatedAt": updatedAt}
	for key, value := range scenarioBaseWatermarkResponse(baseEventWatermark) {
		response[key] = value
	}
	c.JSON(201, response)
}

func (s *Server) getScenario(c *gin.Context) {
	if _, ok := s.canScenario(c, c.Param("id"), "Viewer"); !ok {
		return
	}
	record, err := s.loadScenario(c, c.Param("id"))
	if err != nil {
		fail(c, 404, err)
		return
	}
	c.JSON(200, scenarioResponse(record, true))
}

func (s *Server) appendScenarioEvent(c *gin.Context) {
	scenarioID := c.Param("id")
	if _, ok := s.canScenario(c, scenarioID, "Member"); !ok {
		return
	}
	var input struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if bind(c, &input) {
		return
	}
	input.Type = strings.TrimSpace(input.Type)
	tx, err := s.db.Begin(c)
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer tx.Rollback(c)
	var projectID string
	var snapshotBytes []byte
	if err := tx.QueryRow(c, "select project_id::text,base_snapshot from scenarios where id=$1 for update", scenarioID).Scan(&projectID, &snapshotBytes); err != nil {
		fail(c, 404, err)
		return
	}
	var snapshot ScenarioSnapshot
	if err := json.Unmarshal(snapshotBytes, &snapshot); err != nil {
		fail(c, 500, err)
		return
	}
	rows, err := tx.Query(c, "select id::text,sequence,event_type,payload,created_by::text,created_at from scenario_events where scenario_id=$1 order by sequence", scenarioID)
	if err != nil {
		fail(c, 500, err)
		return
	}
	var history []ScenarioEventRecord
	for rows.Next() {
		var event ScenarioEventRecord
		if err := rows.Scan(&event.ID, &event.Sequence, &event.EventType, &event.Payload, &event.CreatedBy, &event.CreatedAt); err != nil {
			rows.Close()
			fail(c, 500, err)
			return
		}
		history = append(history, event)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(c, 500, err)
		return
	}
	rows.Close()
	record := scenarioRecord{ProjectID: projectID, Snapshot: snapshot, Events: history}
	if err := validateScenarioRecord(record); err != nil {
		fail(c, 500, err)
		return
	}
	state, err := scenarioStateFromRecord(record)
	if err != nil {
		fail(c, 500, err)
		return
	}
	nextSequence := int64(len(history) + 1)
	mutation, normalizedPayload, err := decodeScenarioPayload(input.Type, input.Payload, state)
	_ = mutation
	if err != nil {
		writeScenarioError(c, err)
		return
	}
	candidate := state.Clone()
	candidateEvent := ScenarioEventRecord{EventType: input.Type, Payload: normalizedPayload}
	if err := candidate.Apply(candidateEvent); err != nil {
		writeScenarioError(c, err)
		return
	}
	eventID := uuid.NewString()
	var createdAt time.Time
	if err := tx.QueryRow(c, `insert into scenario_events(id,scenario_id,sequence,event_type,payload,created_by) values($1,$2,$3,$4,$5,$6) returning created_at`, eventID, scenarioID, nextSequence, input.Type, normalizedPayload, userID(c)).Scan(&createdAt); err != nil {
		fail(c, 400, err)
		return
	}
	if _, err := tx.Exec(c, "update scenarios set updated_at=now() where id=$1", scenarioID); err != nil {
		fail(c, 500, err)
		return
	}
	if err := tx.Commit(c); err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(201, scenarioEventResponse{ID: eventID, Sequence: nextSequence, Type: input.Type, Payload: normalizedPayload, CreatedBy: userID(c), CreatedAt: createdAt})
}

func writeScenarioError(c *gin.Context, err error) {
	var validation scenarioValidationError
	if errors.As(err, &validation) {
		code := 400
		if validation.Conflict {
			code = 409
		}
		body := gin.H{"error": validation.Message}
		if len(validation.CycleTaskIDs) > 0 {
			body["cycleTaskIds"] = validation.CycleTaskIDs
		}
		c.JSON(code, body)
		return
	}
	if errors.Is(err, errScenarioCycle) {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, errScenarioTaskNotFound) || errors.Is(err, errScenarioMemberNotFound) || errors.Is(err, errScenarioDependencyNotFound) {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	fail(c, 500, err)
}

func (s *Server) scenarioAnalysis(c *gin.Context) {
	if _, ok := s.canScenario(c, c.Param("id"), "Viewer"); !ok {
		return
	}
	record, err := s.loadScenario(c, c.Param("id"))
	if err != nil {
		fail(c, 404, err)
		return
	}
	state, err := scenarioStateFromRecord(record)
	if err != nil {
		fail(c, 500, err)
		return
	}
	analysis, err := AnalyzeScenario(state, strings.TrimSpace(c.Query("asOf")))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"scenario": scenarioResponse(record, false), "analysis": analysis})
}

func (s *Server) compareScenario(c *gin.Context) {
	scenarioID := c.Param("id")
	if _, ok := s.canScenario(c, scenarioID, "Viewer"); !ok {
		return
	}
	rightRecord, err := s.loadScenario(c, scenarioID)
	if err != nil {
		fail(c, 404, err)
		return
	}
	right, err := scenarioStateFromRecord(rightRecord)
	if err != nil {
		fail(c, 500, err)
		return
	}
	leftLabel := "base"
	left := newScenarioState(rightRecord.Snapshot)
	againstID := strings.TrimSpace(c.Query("against"))
	if againstID != "" {
		if _, ok := s.canScenario(c, againstID, "Viewer"); !ok {
			return
		}
		leftRecord, loadErr := s.loadScenario(c, againstID)
		if loadErr != nil {
			fail(c, 404, loadErr)
			return
		}
		if leftRecord.ProjectID != rightRecord.ProjectID {
			c.JSON(400, gin.H{"error": "scenarios must belong to the same project"})
			return
		}
		leftLabel = leftRecord.Name
		left, err = scenarioStateFromRecord(leftRecord)
		if err != nil {
			fail(c, 500, err)
			return
		}
	}
	comparison, err := CompareScenarioStates(left, right)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(200, gin.H{"scenario": scenarioResponse(rightRecord, false), "against": gin.H{"id": againstID, "label": leftLabel, "digest": comparison.LeftDigest}, "comparison": comparison})
}
