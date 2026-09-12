package dict

// Get returns the value for key or def if the key is absent.
func Get(m map[string]any, key string, def any) any {
	if v, ok := m[key]; ok {
		return v
	}
	return def
}

// GetOpt returns the value for key and whether it was present, mirroring
// Rayo's optional `.get()` semantics.
func GetOpt(m map[string]any, key string) (any, bool) {
	v, ok := m[key]
	return v, ok
}

// Set sets the value for key.
func Set(m map[string]any, key string, val any) {
	m[key] = val
}

// Has reports whether key is present.
func Has(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// Keys returns the map keys (unordered).
func Keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Merge returns a new map with b's entries overlaid on a's.
func Merge(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// ShallowCopy returns a top-level copy of m; nested containers are shared.
func ShallowCopy(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// DeepCopy returns a fully independent copy of m, recursing into nested
// map[string]any and []any values so mutations to the copy never affect the
// original. Other value types are copied by value (they are treated as
// immutable scalars).
func DeepCopy(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return DeepCopy(x)
	case []any:
		cp := make([]any, len(x))
		for i, e := range x {
			cp[i] = deepCopyValue(e)
		}
		return cp
	default:
		return x
	}
}
