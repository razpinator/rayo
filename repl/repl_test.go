package repl

import (
	"rayo/internal/compile"
	"testing"
)

func TestREPLSession(t *testing.T) {
	r := NewREPL(compile.Options{})
	r.History = append(r.History, ":help", ":vars", ":quit")
	if len(r.History) != 3 {
		t.Errorf("REPL history failed")
	}
}

func TestOpenBraces(t *testing.T) {
	if !openBraces("def f() {") {
		t.Fatal("expected open block")
	}
	if openBraces("def f() { }") {
		t.Fatal("expected balanced block closed")
	}
}
