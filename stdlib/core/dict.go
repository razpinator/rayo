package core

func DictKeys(m map[string]any) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

func DictValues(m map[string]any) []any {
    vals := make([]any, 0, len(m))
    for _, v := range m {
        vals = append(vals, v)
    }
    return vals
}

func DictItems(m map[string]any) [][2]any {
    items := make([][2]any, 0, len(m))
    for k, v := range m {
        items = append(items, [2]any{k, v})
    }
    return items
}
