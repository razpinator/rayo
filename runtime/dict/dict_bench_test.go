package dict

import (
	"testing"
)

func BenchmarkDictGetSet(b *testing.B) {
	m := make(map[string]any)
	for i := 0; i < 1000; i++ {
		m[string(rune(i))] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 1000))
		Set(m, key, i)
		_ = Get(m, key, nil)
	}
}

func BenchmarkDictMerge(b *testing.B) {
	a := make(map[string]any)
	bm := make(map[string]any)
	for i := 0; i < 1000; i++ {
		a[string(rune(i))] = i
		bm[string(rune(i+1000))] = i + 1000
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Merge(a, bm)
	}
}
