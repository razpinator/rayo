package err

import (
	"errors"
	"testing"
)

func TestWrapAndCause(t *testing.T) {
	base := errors.New("base")
	wrapped := Wrap(base, "context")
	if wrapped.Error() != "context: base" {
		t.Errorf("Wrap message: got %q", wrapped.Error())
	}
	if Cause(wrapped).Error() != "base" {
		t.Errorf("Cause failed: got %q", Cause(wrapped).Error())
	}
}

func TestWrapNil(t *testing.T) {
	if Wrap(nil, "ctx") != nil {
		t.Errorf("wrapping nil should yield nil")
	}
}

func TestIsMatchesChain(t *testing.T) {
	sentinel := errors.New("not found")
	wrapped := Wrap(sentinel, "lookup failed")
	if !Is(wrapped, sentinel) {
		t.Errorf("Is should match a sentinel through the wrap chain")
	}
	other := errors.New("different")
	if Is(wrapped, other) {
		t.Errorf("Is should not match an unrelated error")
	}
}

func TestAsFindsType(t *testing.T) {
	base := &customErr{code: 42}
	wrapped := Wrap(base, "ctx")
	var target *customErr
	if !As(wrapped, &target) || target.code != 42 {
		t.Errorf("As should extract the custom error through the chain")
	}
}

type customErr struct{ code int }

func (e *customErr) Error() string { return "custom" }
