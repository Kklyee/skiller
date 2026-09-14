package version

import "testing"

func TestString(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = oldVersion, oldCommit, oldDate
	})
	Version = "1.2.3"
	Commit = "abc123"
	Date = "2026-09-14"

	got := String()
	want := "skiller 1.2.3\ncommit: abc123\nbuilt: 2026-09-14"
	if got != want {
		t.Fatalf("version string: got %q, want %q", got, want)
	}
}
