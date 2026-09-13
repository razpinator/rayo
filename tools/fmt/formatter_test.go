package fmt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFormat_Idempotent verifies fmt(fmt(x)) == fmt(x) across a range of
// statement and expression forms.
func TestFormat_Idempotent(t *testing.T) {
	cases := []string{
		"if{x+1==2:foo}",
		`var x = 42`,
		`def main(){print("hi")}`,
		"data = [1,2,3]",
		`d = {"a":1,"b":2}`,
		"for k,v in items {print(k)}",
		"try{risky()}except ValueError as e{print(e)}finally{cleanup()}",
		"if a{b()}elif c{d()}else{e()}",
		"x = obj.field.method(1,2)",
		"y = arr[0][1]",
		"z = a and b or not c",
		"# a leading comment\nvar x = 1 # trailing\n",
		"while true{step()}",
	}
	for _, src := range cases {
		once := Format(src)
		twice := Format(once)
		if once != twice {
			t.Errorf("not idempotent for %q:\n--- once ---\n%s\n--- twice ---\n%s", src, once, twice)
		}
	}
}

// TestFormat_Snapshots formats the bundled example programs and asserts the
// output is stable (idempotent) and matches the checked-in golden snapshot in
// testdata. Run with -update to regenerate snapshots.
func TestFormat_Snapshots(t *testing.T) {
	examples := []string{
		"group_by.ryo",
		"try_except.ryo",
		"kitchen_sink.ryo",
	}
	for _, name := range examples {
		name := name
		t.Run(name, func(t *testing.T) {
			srcPath := filepath.Join("testdata", "in", name)
			src, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("read input %s: %v", srcPath, err)
			}
			got := Format(string(src))

			// Idempotency on the real example.
			if again := Format(got); again != got {
				t.Errorf("formatter not idempotent on %s", name)
			}

			goldenPath := filepath.Join("testdata", "golden", name+".fmt")
			if os.Getenv("UPDATE_SNAPSHOTS") == "1" {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read snapshot %s: %v (run with UPDATE_SNAPSHOTS=1 to create)", goldenPath, err)
			}
			if got != string(want) {
				t.Errorf("snapshot mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}

// TestFormat_Indentation checks nested blocks indent by four spaces per level.
func TestFormat_Indentation(t *testing.T) {
	src := "def f(){if x{return y}}"
	got := Format(src)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	// Expect a return line indented two levels (8 spaces).
	found := false
	for _, ln := range lines {
		if strings.Contains(ln, "return") {
			if strings.HasPrefix(ln, indentUnit+indentUnit) {
				found = true
			} else {
				t.Errorf("return not indented two levels: %q", ln)
			}
		}
	}
	if !found {
		t.Errorf("no return line found in:\n%s", got)
	}
}
