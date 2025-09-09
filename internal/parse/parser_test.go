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
    }
}
