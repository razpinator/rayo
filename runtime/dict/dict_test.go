package dict

import "testing"

func TestDictHelpers(t *testing.T) {
    m := map[string]any{"a": 1}
    if Get(m, "a", 0) != 1 {
        t.Errorf("Get failed")
    }
    if Get(m, "b", 42) != 42 {
        t.Errorf("Get default failed")
    }
    Set(m, "b", 99)
    if m["b"] != 99 {
        t.Errorf("Set failed")
    }
    m2 := map[string]any{"c": 3}
    merged := Merge(m, m2)
    if merged["c"] != 3 || merged["a"] != 1 {
        t.Errorf("Merge failed")
    }
    copy := DeepCopy(m)
    if copy["a"] != 1 {
        t.Errorf("DeepCopy failed")
    }
}
