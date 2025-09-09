package dict

// Get returns the value for key or default.
func Get(m map[string]any, key string, def any) any {
    v, ok := m[key]
    if ok {
        return v
    }
    return def
}

// Set sets the value for key.
func Set(m map[string]any, key string, val any) {
    m[key] = val
}

// Merge merges two maps.
func Merge(a, b map[string]any) map[string]any {
    out := make(map[string]any)
    for k, v := range a {
        out[k] = v
    }
    for k, v := range b {
        out[k] = v
    }
    return out
}

// DeepCopy returns a shallow copy (for demo; deep copy for nested maps can be added).
func DeepCopy(m map[string]any) map[string]any {
    out := make(map[string]any)
    for k, v := range m {
        out[k] = v
    }
    return out
}
