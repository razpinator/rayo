package core

import (
	"errors"
	"testing"
)

func TestRaiseWrapCause(t *testing.T) {
	e := Raise("fail")
	if e.Error() != "fail" {
		t.Errorf("Raise failed: %q", e.Error())
	}
	e2 := Wrap(e, "context")
	if e2.Error() != "context: fail" {
		t.Errorf("Wrap message: %q", e2.Error())
	}
	if Cause(e2).Error() != "fail" {
		t.Errorf("Cause failed: %q", Cause(e2).Error())
	}
}

func TestRaisefFormats(t *testing.T) {
	e := Raisef("code %d: %s", 42, "boom")
	if e.Error() != "code 42: boom" {
		t.Errorf("Raisef failed: %q", e.Error())
	}
}

func TestIsMatchesSentinelThroughChain(t *testing.T) {
	sentinel := Raise("not found")
	wrapped := Wrap(sentinel, "lookup")
	if !Is(wrapped, sentinel) {
		t.Errorf("Is should match sentinel through wrap")
	}
	if Is(wrapped, errors.New("other")) {
		t.Errorf("Is should not match an unrelated error")
	}
}

func TestAsExtractsCustom(t *testing.T) {
	base := &appErr{code: 7}
	wrapped := Wrap(base, "ctx")
	var target *appErr
	if !As(wrapped, &target) || target.code != 7 {
		t.Errorf("As should extract custom error through wrap")
	}
}

func TestWithStack(t *testing.T) {
	e := Raise("fail")
	e2 := WithStack(e)
	if e2 == nil || e2.Error() == e.Error() {
		t.Errorf("WithStack failed to add stack trace")
	}
	if !Is(e2, e) {
		t.Errorf("WithStack should preserve the original error in the chain")
	}
	if WithStack(nil) != nil {
		t.Errorf("WithStack(nil) should be nil")
	}
}

type appErr struct{ code int }

func (e *appErr) Error() string { return "app error" }
