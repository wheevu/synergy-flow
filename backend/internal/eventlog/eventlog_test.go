package eventlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendReadAfterAndRecover(t *testing.T) {
	root := t.TempDir()
	l, err := Open(root, true)
	if err != nil {
		t.Fatal(err)
	}

	id1, err := l.Append("project-1", []byte(`{"type":"task.created"}`))
	if err != nil {
		t.Fatal(err)
	}
	id2, err := l.Append("project-1", []byte(`{"type":"task.moved"}`))
	if err != nil {
		t.Fatal(err)
	}
	if id1 != 1 || id2 != 2 {
		t.Fatalf("ids = %d,%d; want 1,2", id1, id2)
	}

	recs, err := l.ReadAfter("project-1", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].ID != 2 || string(recs[0].Payload) != `{"type":"task.moved"}` {
		t.Fatalf("unexpected replay: %#v", recs)
	}

	reopened, err := Open(root, false)
	if err != nil {
		t.Fatal(err)
	}
	id3, err := reopened.Append("project-1", []byte(`{"type":"task.updated"}`))
	if err != nil {
		t.Fatal(err)
	}
	if id3 != 3 {
		t.Fatalf("id after recovery = %d, want 3", id3)
	}
}

func TestRecoverTruncatesCorruptTail(t *testing.T) {
	root := t.TempDir()
	l, err := Open(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Append("project-1", []byte(`ok`)); err != nil {
		t.Fatal(err)
	}
	seg := filepath.Join(root, "projects", "project-1", "0000000000000001.seg")
	f, err := os.OpenFile(seg, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	reopened, err := Open(root, false)
	if err != nil {
		t.Fatal(err)
	}
	recs, err := reopened.ReadAfter("project-1", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || string(recs[0].Payload) != "ok" {
		t.Fatalf("records after corrupt-tail recovery = %#v", recs)
	}
}

func TestRejectsUnsafeProjectID(t *testing.T) {
	l, err := Open(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Append("../escape", []byte("bad")); err == nil {
		t.Fatal("expected unsafe project id error")
	}
}
