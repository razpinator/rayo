package core

import (
    "errors"
    "testing"
)

func TestRaiseWrapCauseIs(t *testing.T) {
    e := Raise("fail")
    if e.Error() != "fail" {
        t.Errorf("Raise failed")
    }
    e2 := Wrap(e, "context")
    if Cause(e2).Error() != "fail" {
        t.Errorf("Cause failed")
    }
    if !Is(e, errors.New("other")) {
        t.Errorf("Is failed for same type")
    }
}

func TestWithStack(t *testing.T) {
    e := Raise("fail")
    e2 := WithStack(e)
    if e2 == nil || e2.Error() == e.Error() {
        t.Errorf("WithStack failed to add stack trace")
    }
}
