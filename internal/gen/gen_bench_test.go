package gen

import (
	"os"
	"path/filepath"
	"rayo/internal/parse"
	"testing"
)

func BenchmarkCodegenLargeFile(b *testing.B) {
	srcPath := filepath.Join("..", "..", "testdata", "golden", "blocks_if_while_for.ryo")
	src, err := os.ReadFile(srcPath)
	if err != nil {
		b.Fatalf("failed to read source file: %v", err)
	}
	parser := parse.NewParser(string(src))
	mod := parser.ParseModule()
	ctx := NewGenContext("main")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EmitModule(mod, ctx)
	}
}
