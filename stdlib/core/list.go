package core

func Map[T any, U any](list []T, fn func(T) U) []U {
    out := make([]U, len(list))
    for i, v := range list {
        out[i] = fn(v)
    }
    return out
}

func Filter[T any](list []T, fn func(T) bool) []T {
    out := []T{}
    for _, v := range list {
        if fn(v) {
            out = append(out, v)
        }
    }
    return out
}

func Reduce[T any, U any](list []T, init U, fn func(U, T) U) U {
    acc := init
    for _, v := range list {
        acc = fn(acc, v)
    }
    return acc
}
