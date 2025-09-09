package data

// Map applies fn to each element of list.
func Map[T any, U any](list []T, fn func(T) U) []U {
    out := make([]U, len(list))
    for i, v := range list {
        out[i] = fn(v)
    }
    return out
}

// Filter returns elements where fn returns true.
func Filter[T any](list []T, fn func(T) bool) []T {
    out := []T{}
    for _, v := range list {
        if fn(v) {
            out = append(out, v)
        }
    }
    return out
}

// Reduce folds list with fn and initial value.
func Reduce[T any, U any](list []T, init U, fn func(U, T) U) U {
    acc := init
    for _, v := range list {
        acc = fn(acc, v)
    }
    return acc
}

// Agg aggregates list with fn.
func Agg[T any, U any](list []T, fn func(U, T) U, init U) U {
    return Reduce(list, init, fn)
}

// GroupBy groups list by key_fn.
func GroupBy[T any, K comparable](list []T, keyFn func(T) K) map[K][]T {
    out := map[K][]T{}
    for _, v := range list {
        k := keyFn(v)
        out[k] = append(out[k], v)
    }
    return out
}

// Select returns list of dicts with selected fields.
func Select(list []map[string]any, fields []string) []map[string]any {
    out := []map[string]any{}
    for _, row := range list {
        m := map[string]any{}
        for _, f := range fields {
            m[f] = row[f]
        }
        out = append(out, m)
    }
    return out
}
