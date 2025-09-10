package data

import (
	"testing"
)

type TestItem struct {
	ID   int
	Name string
}

func BenchmarkGroupBy(b *testing.B) {
	list := make([]TestItem, 10000)
	for i := 0; i < 10000; i++ {
		list[i] = TestItem{ID: i % 100, Name: string(rune(i))}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GroupBy(list, func(item TestItem) int { return item.ID })
	}
}
