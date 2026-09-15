package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileProfile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ConfigFileName)
	writeConfig(t, path, "profile = \"go-backend\"\n")

	config, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load profile config: %v", err)
	}
	if config.Profile != "go-backend" || len(config.Skills) != 0 {
		t.Fatalf("config: %#v", config)
	}
}

func TestLoadFileSkills(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ConfigFileName)
	writeConfig(t, path, "skills = [\"code-review\", \"tdd\"]\n")

	config, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load skills config: %v", err)
	}
	if config.Profile != "" || !sameStrings(config.Skills, []string{"code-review", "tdd"}) {
		t.Fatalf("config: %#v", config)
	}
}

func TestLoadFileSupportsProjectOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	writeConfig(t, path, "profile = \"backend\"\ninclude = [\"research\"]\nexclude = [\"wizard\"]\n")

	config, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load project overrides: %v", err)
	}
	if config.Profile != "backend" || !sameStrings(config.Include, []string{"research"}) || !sameStrings(config.Exclude, []string{"wizard"}) {
		t.Fatalf("config: %#v", config)
	}
}

func TestLoadFileRejectsDuplicateProjectOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	writeConfig(t, path, "skills = [\"code-review\"]\ninclude = [\"research\", \"research\"]\n")

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "include skill") {
		t.Fatalf("duplicate include error: %v", err)
	}
}

func TestLoadFileRejectsInvalidSources(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "none", data: "", want: "exactly one"},
		{name: "both", data: "profile = \"backend\"\nskills = [\"tdd\"]\n", want: "exactly one"},
		{name: "duplicate", data: "skills = [\"tdd\", \"tdd\"]\n", want: "listed more than once"},
		{name: "empty skill", data: "skills = [\"\"]\n", want: "must not be empty"},
		{name: "unknown", data: "profile = \"backend\"\nother = true\n", want: "unsupported field"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ConfigFileName)
			writeConfig(t, path, test.data)
			_, err := LoadFile(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error: %v, want %q", err, test.want)
			}
		})
	}
}

func TestFindWalksToNearestProjectConfig(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("create child: %v", err)
	}
	parentConfig := filepath.Join(root, ConfigFileName)
	childConfig := filepath.Join(filepath.Dir(child), ConfigFileName)
	writeConfig(t, parentConfig, "profile = \"root\"\n")
	writeConfig(t, childConfig, "profile = \"service\"\n")

	path, err := Find(child)
	if err != nil {
		t.Fatalf("find config: %v", err)
	}
	if path != childConfig {
		t.Fatalf("config path: got %q, want %q", path, childConfig)
	}
	config, loadedPath, err := Load(child)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if config.Profile != "service" || loadedPath != childConfig {
		t.Fatalf("loaded config: %#v at %q", config, loadedPath)
	}
}

func TestFindReportsMissingConfig(t *testing.T) {
	_, err := Find(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error: %v", err)
	}
}

func writeConfig(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
