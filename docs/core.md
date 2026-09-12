# Rayo Core Library

## Strings
- `StrLen(s string) int` — Length of string
- `StrUpper(s string) string` — Uppercase
- `StrLower(s string) string` — Lowercase
- `StrSplit(s, sep string) []string` — Split string

## Math
- `Abs(x float64) float64` — Absolute value
- `Pow(x, y float64) float64` — Power
- `Max(x, y float64) float64` — Maximum
- `Min(x, y float64) float64` — Minimum

## List
- `Map(list, fn)` — Map over list
- `Filter(list, fn)` — Filter list
- `Reduce(list, init, fn)` — Reduce list

## Dict
- `DictKeys(m map[string]any) []string` — Keys
- `DictValues(m map[string]any) []any` — Values
- `DictItems(m map[string]any) [][2]any` — Items

## Time
- `Now() time.Time` — Current time
- `FormatTime(t, layout)` — Format time
- `ParseTime(layout, value)` — Parse time

## Errors
- `Raise(msg string) error` — New error from literal text
- `Raisef(format string, args...) error` — New error from a format string
- `Wrap(err error, msg string) error` — Wrap while preserving the chain (`nil`-safe)
- `Cause(err error) error` — Root cause after fully unwrapping
- `Unwrap(err error) error` — One level of unwrapping
- `Is(err, target error) bool` — Standard `errors.Is` chain matching
- `As(err error, target any) bool` — Standard `errors.As` extraction
- `WithStack(err error) error` — Attach a stack trace, keeping the chain intact

> `Is`/`As` use standard-library semantics; matching is by value identity and
> custom `Is`/`As` methods through the wrap chain, not by comparing type names.

## Runtime comparison (`runtime/core`)
- `Compare(a, b) int` — Order two values (`int`, `float`, `str`); mixed
  int/float operands are compared numerically
- `Equal(a, b) bool` — Value equality across numeric and string kinds
- `Truthy(v) bool` — Rayo truthiness (nil/false/0/"" are falsy)

---

See unit tests for usage examples.
