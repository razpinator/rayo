package parse

import (
    "testing"
)

func TestParser_ParseModule(t *testing.T) {
    src := "import 'core'\nvar x = 42"
    p := NewParser(src)
    mod := p.ParseModule()
    if mod == nil || len(mod.Imports) != 1 || len(mod.Body) != 1 {
        t.Errorf("parser failed to parse module structure")
        // Print tokens for debugging
        lx2 := NewParser(src).lx
        t.Log("Tokens:")
        for {
            tok := lx2.Next()
            if tok.Kind == 0 { // TokenEOF
                break
            }
            t.Logf("%v: %q", tok.Kind, tok.Value)
        }
        // Print parser errors
        t.Logf("Parser errors: %v", p.errors)
        // Print parsed AST
        t.Logf("Parsed AST: %+v", mod)
        if mod != nil {
            t.Logf("Imports: %+v", mod.Imports)
            t.Logf("Body: %+v", mod.Body)
        }
    }
}
