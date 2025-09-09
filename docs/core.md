# Functure Core Library

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

---

See unit tests for usage examples.
