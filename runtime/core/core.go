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

// Compare orders two values of the same comparable kind, returning -1, 0, or 1.
// It handles int, float64, and string operands, and coerces int/float mixes to
// float so numeric comparisons across the two behave as users expect. Values of
// incomparable or mismatched kinds compare as equal (0).
func Compare(a, b Any) int {
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			switch {
			case af < bf:
				return -1
			case af > bf:
				return 1
			default:
				return 0
			}
		}
	}
	if x, ok := a.(string); ok {
		if y, ok := b.(string); ok {
			switch {
			case x < y:
				return -1
			case x > y:
				return 1
			default:
				return 0
			}
		}
	}
	return 0 // fallback for mismatched or incomparable kinds
}

// toFloat converts numeric values to float64 for uniform comparison.
func toFloat(v Any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case float32:
		return float64(x), true
	default:
		return 0, false
	}
}

// Equal reports value equality across the numeric and string kinds Compare
// understands; other kinds use Go's == via interface comparison.
func Equal(a, b Any) bool {
	if _, ok := toFloat(a); ok {
		if _, ok := toFloat(b); ok {
			return Compare(a, b) == 0
		}
	}
	return a == b
}
