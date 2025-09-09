package core

import "testing"

func TestOption(t *testing.T) {
    o := Some(42)
    if !o.IsSome() || o.Unwrap() != 42 {
        t.Errorf("Option failed")
    }
    n := None[int]()
    if n.IsSome() {
        t.Errorf("None should not be Some")
    }
}

func TestTruthy(t *testing.T) {
    if Truthy(nil) {
        t.Errorf("nil should not be truthy")
    }
    if !Truthy(1) {
        t.Errorf("1 should be truthy")
    }
    if Truthy(0) {
        t.Errorf("0 should not be truthy")
    }
    if !Truthy("foo") {
        t.Errorf("non-empty string should be truthy")
    }
    if Truthy("") {
        t.Errorf("empty string should not be truthy")
    }
}

func TestCompare(t *testing.T) {
    if Compare(1, 2) != -1 || Compare(2, 1) != 1 || Compare(2, 2) != 0 {
        t.Errorf("int compare failed")
    }
    if Compare("a", "b") != -1 || Compare("b", "a") != 1 || Compare("a", "a") != 0 {
        t.Errorf("string compare failed")
    }
}
