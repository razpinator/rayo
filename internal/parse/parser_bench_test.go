package parse

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkParseLargeFile(b *testing.B) {
	srcPath := filepath.Join("..", "..", "testdata", "golden", "blocks_if_while_for.ryo")
	src, err := os.ReadFile(srcPath)
	if err != nil {
		b.Fatalf("failed to read source file: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(string(src))
		_ = parser.ParseModule()
	}
}
