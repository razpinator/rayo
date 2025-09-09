package data

import "testing"

func TestMapFilterReduceAgg(t *testing.T) {
    xs := []int{1, 2, 3}
    ys := Map(xs, func(x int) int { return x * 2 })
    if ys[0] != 2 || ys[2] != 6 {
        t.Errorf("Map failed")
    }
    zs := Filter(xs, func(x int) bool { return x > 1 })
    if len(zs) != 2 {
        t.Errorf("Filter failed")
    }
    sum := Reduce(xs, 0, func(acc, x int) int { return acc + x })
    if sum != 6 {
        t.Errorf("Reduce failed")
    }
    agg := Agg(xs, func(acc, x int) int { return acc + x }, 0)
    if agg != 6 {
        t.Errorf("Agg failed")
    }
}

func TestGroupBy(t *testing.T) {
    xs := []string{"a", "bb", "c"}
    groups := GroupBy(xs, func(s string) int { return len(s) })
    if len(groups[1]) != 2 || len(groups[2]) != 1 {
        t.Errorf("GroupBy failed")
    }
}

func TestSelect(t *testing.T) {
    rows := []map[string]any{{"a": 1, "b": 2}, {"a": 3, "b": 4}}
    out := Select(rows, []string{"a"})
    if out[0]["a"] != 1 || out[1]["a"] != 3 {
        t.Errorf("Select failed")
    }
}

func TestSelectStructFields(t *testing.T) {
    type Row struct{ A, B int }
    rows := []Row{{1, 2}, {3, 4}}
    out := SelectStructFields(rows, []string{"A"})
    if out[0]["A"] != 1 || out[1]["A"] != 3 {
        t.Errorf("SelectStructFields failed")
    }
}
