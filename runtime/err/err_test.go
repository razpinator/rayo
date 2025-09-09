package err

import (
    "errors"
    "testing"
)

func TestWrapAndCause(t *testing.T) {
    base := errors.New("base")
    wrapped := Wrap(base, "context")
    if Cause(wrapped).Error() != "base" {
        t.Errorf("Cause failed")
    }
}

func TestIs(t *testing.T) {
    e1 := errors.New("foo")
    e2 := errors.New("bar")
    if !Is(e1, e2) {
        t.Errorf("Is failed for same type")
    }
}
