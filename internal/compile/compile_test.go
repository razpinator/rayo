package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportCycle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.ryo"), "import './b.ryo'\nprint(1)\n")
	writeFile(t, filepath.Join(dir, "b.ryo"), "import './a.ryo'\nprint(2)\n")
	_, err := BuildProgram(filepath.Join(dir, "a.ryo"), Options{})
	if err == nil || !strings.Contains(err.Error(), "import cycle") {
		t.Fatalf("expected import cycle error, got %v", err)
	}
}

func TestMissingImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ryo"), "import './nope.ryo'\nprint(1)\n")
	_, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err == nil || !strings.Contains(err.Error(), "cannot resolve import") {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestIncludePath(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(libDir, "x.ryo"), "def n() {\nreturn 9\n}\n")
	writeFile(t, filepath.Join(dir, "main.ryo"), "import './x.ryo'\nprint(n())\n")
	_, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{IncludePaths: []string{libDir}})
	if err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
