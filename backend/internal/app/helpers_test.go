package app

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRank(t *testing.T) {
	tests := []struct {
		role string
		want int
	}{
		{"Viewer", 1},
		{"Member", 2},
		{"Admin", 3},
		{"Owner", 4},
		{"Unknown", 0},
	}
	for _, tt := range tests {
		if got := rank(tt.role); got != tt.want {
			t.Errorf("rank(%q) = %d, want %d", tt.role, got, tt.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"  Spaces Around  ", "spaces-around"},
		{"UPPERCASE", "uppercase"},
		{"special!@#chars", "special---chars"},
		{"already-kebab", "already-kebab"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := slugify(tt.input); got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHashToken(t *testing.T) {
	h := hashToken("test-token-123")
	if len(h) != 64 { // SHA-256 hex is 64 chars
		t.Errorf("hashToken() = %q (len=%d), want len=64", h, len(h))
	}
	// Same input produces same hash
	h2 := hashToken("test-token-123")
	if h != h2 {
		t.Errorf("hashToken() not deterministic: %q vs %q", h, h2)
	}
	// Different inputs produce different hashes
	h3 := hashToken("different-token")
	if h == h3 {
		t.Errorf("hashToken() produced same hash for different inputs")
	}
}

func TestRandString(t *testing.T) {
	tests := []int{0, 1, 8, 16, 32, 64}
	for _, n := range tests {
		s := randString(n)
		if len(s) != n {
			t.Errorf("randString(%d) = %q (len=%d), want len=%d", n, s, len(s), n)
		}
	}
	// Ensure randomness
	s1 := randString(16)
	s2 := randString(16)
	if s1 == s2 {
		t.Errorf("randString() returned same value twice: %q", s1)
	}
}

func TestMax(t *testing.T) {
	tests := []struct{ a, b, want int }{
		{1, 2, 2},
		{5, 3, 5},
		{0, 0, 0},
		{-1, 1, 1},
		{100, 100, 100},
	}
	for _, tt := range tests {
		if got := max(tt.a, tt.b); got != tt.want {
			t.Errorf("max(%d,%d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestStrp(t *testing.T) {
	m := map[string]any{
		"name": "test",
		"age":  30,
	}
	if got := strp(m, "name"); got == nil || *got != "test" {
		t.Errorf("strp(name) = %v, want 'test'", got)
	}
	if got := strp(m, "age"); got != nil {
		t.Errorf("strp(age) = %v, want nil", got)
	}
	if got := strp(m, "missing"); got != nil {
		t.Errorf("strp(missing) = %v, want nil", got)
	}
}

func TestStringSlice(t *testing.T) {
	cases := []struct {
		input any
		want  []string
	}{
		{[]any{"a", "b", "c"}, []string{"a", "b", "c"}},
		{[]any{42, true}, []string{"42", "true"}}, // non-string values are fmt.Sprint'd
		{[]any{}, []string{}},
		{"not a slice", nil},
		{nil, nil},
	}
	for _, c := range cases {
		got := stringSlice(c.input)
		if len(got) != len(c.want) {
			t.Errorf("stringSlice(%v) = %v, want %v", c.input, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("stringSlice(%v) = %v, want %v", c.input, got, c.want)
				break
			}
		}
	}
}

// TestEventSerialization verifies the canonical SSE event payload shape.
// The frontend SSE consumer does NOT parse event JSON - it only acts on any
// arriving event as an invalidation trigger. But the payload shape is still
// pinned by this test to prevent accidental regressions.
func TestEventSerialization(t *testing.T) {
	e := Event{
		Type:      "task.moved",
		ProjectID: "proj-123",
		ActorID:   "user-456",
		Data:      map[string]any{"id": "task-789", "columnId": "col-abc", "position": 3},
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)

	// Must contain all expected fields
	for _, field := range []string{`"type":"task.moved"`, `"projectId":"proj-123"`, `"actorId":"user-456"`, `"data"`, `"id":"task-789"`} {
		if !contains(got, field) {
			t.Errorf("Event JSON missing %q: %s", field, got)
		}
	}

	// Must NOT contain old buggy field shapes
	if contains(got, `"ProjectID"`) || contains(got, `"ActorID"`) {
		t.Errorf("Event JSON contains Go field name (not JSON tag): %s", got)
	}

	// Verify the JSON can be unmarshalled back
	var decoded Event
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Type != "task.moved" {
		t.Errorf("decoded.Type = %q, want %q", decoded.Type, "task.moved")
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestUuidFromBytes(t *testing.T) {
	b := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	got := uuidFromBytes(b)
	if len(got) != 36 {
		t.Errorf("uuidFromBytes() = %q (len=%d), expected len=36", got, len(got))
	}
	// Check the format matches UUID pattern xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for i, c := range got {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				t.Errorf("uuidFromBytes() = %q, expected '-' at position %d", got, i)
			}
		}
	}
}

func TestWriteSSEIncludesReplayID(t *testing.T) {
	var b bytes.Buffer
	writeSSE(&b, 42, "message", []byte(`{"type":"task.updated"}`))
	got := b.String()
	for _, want := range []string{"id: 42\n", "event: message\n", `data: {"type":"task.updated"}`, "\n\n"} {
		if !contains(got, want) {
			t.Fatalf("SSE frame missing %q: %q", want, got)
		}
	}
}

func TestEventAfterPrefersQueryThenHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/events?after=9", nil)
	c.Request.Header.Set("Last-Event-ID", "3")
	if got := eventAfter(c); got != 9 {
		t.Fatalf("eventAfter query = %d, want 9", got)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/events", nil)
	c.Request.Header.Set("Last-Event-ID", "3")
	if got := eventAfter(c); got != 3 {
		t.Fatalf("eventAfter header = %d, want 3", got)
	}
}
