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

func TestDeepCopyIsIndependent(t *testing.T) {
	orig := map[string]any{
		"scalar": 1,
		"nested": map[string]any{"k": "v"},
		"list":   []any{1, map[string]any{"deep": true}},
	}
	cp := DeepCopy(orig)

	// Mutate the copy's nested structures.
	cp["nested"].(map[string]any)["k"] = "changed"
	cp["list"].([]any)[1].(map[string]any)["deep"] = false

	if orig["nested"].(map[string]any)["k"] != "v" {
		t.Errorf("DeepCopy shared a nested map: %v", orig["nested"])
	}
	if orig["list"].([]any)[1].(map[string]any)["deep"] != true {
		t.Errorf("DeepCopy shared a nested list element: %v", orig["list"])
	}
}

func TestShallowCopyShares(t *testing.T) {
	orig := map[string]any{"nested": map[string]any{"k": "v"}}
	cp := ShallowCopy(orig)
	cp["nested"].(map[string]any)["k"] = "changed"
	if orig["nested"].(map[string]any)["k"] != "changed" {
		t.Errorf("ShallowCopy should share nested containers")
	}
}

func TestHasKeysGetOpt(t *testing.T) {
	m := map[string]any{"a": 1}
	if !Has(m, "a") || Has(m, "b") {
		t.Errorf("Has failed")
	}
	if v, ok := GetOpt(m, "a"); !ok || v != 1 {
		t.Errorf("GetOpt present failed")
	}
	if _, ok := GetOpt(m, "missing"); ok {
		t.Errorf("GetOpt absent should be false")
	}
	if len(Keys(m)) != 1 {
		t.Errorf("Keys failed")
	}
}
