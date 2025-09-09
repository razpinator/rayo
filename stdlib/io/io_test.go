package io

import (
    "os"
    "testing"
)

func TestReadWriteText(t *testing.T) {
    path := "test.txt"
    err := WriteText(path, "hello")
    if err != nil {
        t.Fatalf("WriteText failed: %v", err)
    }
    s, err := ReadText(path)
    if err != nil || s != "hello" {
        t.Fatalf("ReadText failed: %v", err)
    }
    os.Remove(path)
}

func TestReadWriteBytes(t *testing.T) {
    path := "test.bin"
    err := WriteBytes(path, []byte{1, 2, 3})
    if err != nil {
        t.Fatalf("WriteBytes failed: %v", err)
    }
    b, err := ReadBytes(path)
    if err != nil || len(b) != 3 {
        t.Fatalf("ReadBytes failed: %v", err)
    }
    os.Remove(path)
}

func TestJSONRoundTrip(t *testing.T) {
    path := "test.json"
    data := map[string]any{"a": 1, "b": "foo"}
    err := DumpJSON(path, data)
    if err != nil {
        t.Fatalf("DumpJSON failed: %v", err)
    }
    var out map[string]any
    err = LoadJSON(path, &out)
    if err != nil || out["a"] != float64(1) || out["b"] != "foo" {
        t.Fatalf("LoadJSON failed: %v", err)
    }
    os.Remove(path)
}

func TestCSVRoundTrip(t *testing.T) {
    path := "test.csv"
    rows := []map[string]string{{"a": "1", "b": "foo"}, {"a": "2", "b": "bar"}}
    err := DumpCSV(path, rows)
    if err != nil {
        t.Fatalf("DumpCSV failed: %v", err)
    }
    out, err := LoadCSV(path)
    if err != nil || len(out) != 2 || out[0]["a"] != "1" || out[1]["b"] != "bar" {
        t.Fatalf("LoadCSV failed: %v", err)
    }
    os.Remove(path)
}
