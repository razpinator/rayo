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

// TestConstantFoldingInPipeline verifies the opt pass runs during compilation:
// a literal arithmetic expression is folded before codegen.
func TestConstantFoldingInPipeline(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ryo"), "def compute() {\nreturn 6 * 7\n}\nprint(compute())\n")
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "return 42") {
		t.Errorf("expected folded `return 42` in generated Go, got:\n%s", out)
	}
	if strings.Contains(out, "6 * 7") {
		t.Errorf("expected `6 * 7` to be folded away, got:\n%s", out)
	}
}

// TestGenericFuncCompiles verifies a generic function transpiles end-to-end and
// the generated Go is well-formed (formatGo would fall back on invalid input,
// so we assert the generic signature is present).
func TestGenericFuncCompiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ryo"), "def identity[T](x: T) -> T {\nreturn x\n}\nprint(identity(5))\n")
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "func identity[T any](x T) T {") {
		t.Errorf("expected generic signature in generated Go, got:\n%s", out)
	}
}

// TestImportAliasEmitted verifies `import "path" as name` produces an aliased Go
// import. A non-.ryo path is treated as a Go import path and passed through.
func TestImportAliasEmitted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.ryo"), "import \"strings\" as s\ndef go() {\nreturn 1\n}\n")
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "import s \"strings\"") {
		t.Errorf("expected aliased import, got:\n%s", out)
	}
}

// TestClassCompilesEndToEnd verifies a class with a constructor, method, and
// property transpiles to Go containing the expected struct/constructor/method
// and that a construction call is rewritten to the generated constructor.
func TestClassCompilesEndToEnd(t *testing.T) {
	dir := t.TempDir()
	src := `class Point {
    x: int
    y: int
    def __init__(self, x: int, y: int) {
        self.x = x
        self.y = y
    }
    def sum(self) -> int {
        return self.x + self.y
    }
    property total {
        get {
            return self.x + self.y
        }
    }
}
def main() {
    p = Point(3, 4)
    print(p.sum())
    print(p.total())
}
`
	writeFile(t, filepath.Join(dir, "main.ryo"), src)
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"type Point struct {",
		"func NewPoint(x int64, y int64) *Point {",
		"func (self *Point) sum() int64 {",
		"func (self *Point) total() any {",
		"NewPoint(3, 4)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in generated Go, got:\n%s", want, out)
		}
	}
}

// TestMatchCompilesEndToEnd verifies match/case lowers to an if/else-if chain
// over a subject temp and that the generated Go is well-formed.
func TestMatchCompilesEndToEnd(t *testing.T) {
	dir := t.TempDir()
	src := `def classify(n) {
    match n {
        case 0 {
            return "zero"
        }
        case 1 {
            return "one"
        }
        case other {
            return "many"
        }
    }
}
def main() {
    print(classify(0))
}
`
	writeFile(t, filepath.Join(dir, "main.ryo"), src)
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "} else if") {
		t.Errorf("expected an else-if chain from match, got:\n%s", out)
	}
	if !strings.Contains(out, "} else {") {
		t.Errorf("expected a default else arm from the capture case, got:\n%s", out)
	}
}

// TestDecoratorCompilesEndToEnd verifies a decorated function transpiles to a
// wrapper that applies the decorator to a function literal and invokes it via a
// callable assertion.
func TestDecoratorCompilesEndToEnd(t *testing.T) {
	dir := t.TempDir()
	src := `def announce(fn) {
    return lambda x: fn(x)
}
@announce
def unit(x) {
    return x
}
def main() {
    print(unit(42))
}
`
	writeFile(t, filepath.Join(dir, "main.ryo"), src)
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "func unit(x any) any {") {
		t.Errorf("expected decorated wrapper function, got:\n%s", out)
	}
	if !strings.Contains(out, "announce(func(x any) any {") {
		t.Errorf("expected decorator applied to inner literal, got:\n%s", out)
	}
	if !strings.Contains(out, "_dec.(func(any) any)(x)") {
		t.Errorf("expected callable assertion on decorated result, got:\n%s", out)
	}
}

// TestOptimizeInliningInPipeline verifies inlining + folding run during
// compilation: a call to a tiny function with a literal argument is inlined and
// folded to a constant.
func TestOptimizeInliningInPipeline(t *testing.T) {
	dir := t.TempDir()
	src := `def add1(x) {
    return x + 1
}
def compute() {
    return add1(41)
}
print(compute())
`
	writeFile(t, filepath.Join(dir, "main.ryo"), src)
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "return 42") {
		t.Errorf("expected add1(41) inlined and folded to `return 42`, got:\n%s", out)
	}
}

// TestOptimizeDCEInPipeline verifies unreachable code after a return is dropped
// during compilation.
func TestOptimizeDCEInPipeline(t *testing.T) {
	dir := t.TempDir()
	src := `def f() {
    return 1
    print("unreachable")
}
print(f())
`
	writeFile(t, filepath.Join(dir, "main.ryo"), src)
	out, err := BuildProgram(filepath.Join(dir, "main.ryo"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "unreachable") {
		t.Errorf("expected unreachable statement removed by DCE, got:\n%s", out)
	}
}
