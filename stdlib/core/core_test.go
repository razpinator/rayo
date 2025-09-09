package core

import (
    "testing"
    "time"
)

func TestStrLen(t *testing.T) {
    if StrLen("abc") != 3 {
        t.Errorf("StrLen failed")
    }
}

func TestStrUpperLower(t *testing.T) {
    if StrUpper("foo") != "FOO" || StrLower("FOO") != "foo" {
        t.Errorf("StrUpper/StrLower failed")
    }
}

func TestStrSplit(t *testing.T) {
    parts := StrSplit("a,b,c", ",")
    if len(parts) != 3 {
        t.Errorf("StrSplit failed")
    }
}

func TestMath(t *testing.T) {
    if Abs(-2) != 2 || Pow(2, 3) != 8 {
        t.Errorf("Math failed")
    }
    if Max(1, 2) != 2 || Min(1, 2) != 1 {
        t.Errorf("Max/Min failed")
    }
}

func TestListOps(t *testing.T) {
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
}

func TestDictOps(t *testing.T) {
    m := map[string]any{"a": 1, "b": 2}
    keys := DictKeys(m)
    if len(keys) != 2 {
        t.Errorf("DictKeys failed")
    }
    vals := DictValues(m)
    if len(vals) != 2 {
        t.Errorf("DictValues failed")
    }
    items := DictItems(m)
    if len(items) != 2 {
        t.Errorf("DictItems failed")
    }
}

func TestTimeOps(t *testing.T) {
    now := Now()
    s := FormatTime(now, time.RFC3339)
    t2, err := ParseTime(time.RFC3339, s)
    if err != nil || !t2.Equal(now) {
        t.Errorf("Time ops failed")
    }
}
