package golden

import (
	"path/filepath"
	"testing"
)

func TestGolden(t *testing.T) {
	dir := filepath.Join("..", "testdata", "golden")
	if err := Run(dir, ""); err != nil {
		t.Fatal(err)
	}
}
