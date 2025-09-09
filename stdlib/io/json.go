package io

import (
    "encoding/json"
    "os"
)

func LoadJSON(path string, v any) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()
    return json.NewDecoder(f).Decode(v)
}

func DumpJSON(path string, v any) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()
    enc := json.NewEncoder(f)
    enc.SetIndent("", "  ")
    return enc.Encode(v)
}
