package obj

// Obj is a dynamic attribute table.
type Obj struct {
    Attrs map[string]any
}

func NewObj() *Obj {
    return &Obj{Attrs: map[string]any{}}
}

func (o *Obj) GetAttr(name string) any {
    return o.Attrs[name]
}

func (o *Obj) SetAttr(name string, val any) {
    o.Attrs[name] = val
}
