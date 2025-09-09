package data

import "reflect"

// SelectStructFields returns a slice of maps with selected fields from a slice of structs.
func SelectStructFields(list any, fields []string) []map[string]any {
    v := reflect.ValueOf(list)
    out := []map[string]any{}
    for i := 0; i < v.Len(); i++ {
        m := map[string]any{}
        elem := v.Index(i)
        for _, f := range fields {
            m[f] = elem.FieldByName(f).Interface()
        }
        out = append(out, m)
    }
    return out
}
