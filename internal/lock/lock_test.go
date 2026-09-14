package lock

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireExcludesSecondProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skiller.lock")

	first, err := Acquire(path)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	defer first.Close()

	if _, err := Acquire(path); err == nil {
		t.Fatal("expected second lock acquisition to fail")
	} else if !strings.Contains(err.Error(), "another Skiller process") {
		t.Fatalf("unexpected lock error: %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("release first lock: %v", err)
	}

	second, err := Acquire(path)
	if err != nil {
		t.Fatalf("acquire lock after release: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("release second lock: %v", err)
	}
}
