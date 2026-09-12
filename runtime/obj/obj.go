package obj

import "strings"

// Obj is a dynamic attribute table backing Rayo objects whose fields are not
// known statically.
type Obj struct {
	Attrs map[string]any
}

func NewObj() *Obj {
	return &Obj{Attrs: map[string]any{}}
}

// GetAttr returns the value of name, or nil if it is unset.
func (o *Obj) GetAttr(name string) any {
	if o == nil || o.Attrs == nil {
		return nil
	}
	return o.Attrs[name]
}

// SetAttr sets name to val.
func (o *Obj) SetAttr(name string, val any) {
	if o.Attrs == nil {
		o.Attrs = map[string]any{}
	}
	o.Attrs[name] = val
}

// HasAttr reports whether name is present.
func (o *Obj) HasAttr(name string) bool {
	if o == nil || o.Attrs == nil {
		return false
	}
	_, ok := o.Attrs[name]
	return ok
}

// SafeGet implements safe navigation (`obj?.attr`): it returns nil instead of
// panicking when the receiver is nil.
func SafeGet(v any, name string) any {
	switch x := v.(type) {
	case nil:
		return nil
	case *Obj:
		return x.GetAttr(name)
	case map[string]any:
		return x[name]
	default:
		return nil
	}
}

// GetPath traverses a dotted path (e.g. "user.profile.email") across nested
// *Obj and map[string]any values. It returns (value, true) when the full path
// resolves and (nil, false) if any segment is missing or nil, modeling chained
// safe navigation without panics.
func GetPath(root any, path string) (any, bool) {
	cur := root
	for _, seg := range strings.Split(path, ".") {
		if seg == "" {
			continue
		}
		switch x := cur.(type) {
		case nil:
			return nil, false
		case *Obj:
			if !x.HasAttr(seg) {
				return nil, false
			}
			cur = x.GetAttr(seg)
		case map[string]any:
			v, ok := x[seg]
			if !ok {
				return nil, false
			}
			cur = v
		default:
			return nil, false
		}
	}
	return cur, true
}

// GetPathOr returns the value at path, or def when the path does not fully
// resolve. This mirrors `root?.a?.b or default` in Rayo.
func GetPathOr(root any, path string, def any) any {
	if v, ok := GetPath(root, path); ok && v != nil {
		return v
	}
	return def
}
