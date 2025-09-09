package repl

import "testing"

func TestREPLSession(t *testing.T) {
    r := NewREPL()
    r.History = append(r.History, ":help", ":vars", ":quit")
    if len(r.History) != 3 {
        t.Errorf("REPL history failed")
    }
    r.Scope["x"] = 42
    if r.Scope["x"] != 42 {
        t.Errorf("REPL scope failed")
    }
}
