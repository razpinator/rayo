package core

// Any is an alias for interface{} in Go.
type Any = interface{}

// Option represents an optional value.
type Option[T any] struct {
    Value *T
}

func None[T any]() Option[T] {
    return Option[T]{Value: nil}
}

func Some[T any](v T) Option[T] {
    return Option[T]{Value: &v}
}

func (o Option[T]) IsSome() bool {
    return o.Value != nil
}

func (o Option[T]) Unwrap() T {
    if o.Value == nil {
        panic("called Unwrap on None")
    }
    return *o.Value
}

func Truthy(v Any) bool {
    switch x := v.(type) {
    case nil:
        return false
    case bool:
        return x
    case int:
        return x != 0
    case string:
        return x != ""
    default:
        return true
    }
}

func Compare(a, b Any) int {
    switch x := a.(type) {
    case int:
        if y, ok := b.(int); ok {
            if x < y {
                return -1
            } else if x > y {
                return 1
            }
            return 0
        }
    case string:
        if y, ok := b.(string); ok {
            if x < y {
                return -1
            } else if x > y {
                return 1
            }
            return 0
        }
    }
    return 0 // fallback
}
