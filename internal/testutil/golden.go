package testutil

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
)

// GoldenCase represents a single golden test case.
type GoldenCase struct {
    Name   string
    Source string
    Expect map[string]string // e.g. {"out": "...", "tokens": "..."}
}

// LoadGoldenCases loads all .pygb files and their expected outputs from a directory.
func LoadGoldenCases(dir string) ([]GoldenCase, error) {
    var cases []GoldenCase
    files, err := ioutil.ReadDir(dir)
    if err != nil {
        return nil, err
    }
    for _, f := range files {
        if strings.HasSuffix(f.Name(), ".pygb") {
            base := strings.TrimSuffix(f.Name(), ".pygb")
            srcPath := filepath.Join(dir, f.Name())
            srcBytes, err := ioutil.ReadFile(srcPath)
            if err != nil {
                return nil, err
            }
            expect := make(map[string]string)
            // Look for .out, .tokens, .ast, .go files
            for _, ext := range []string{".out", ".tokens", ".ast", ".go"} {
                outPath := filepath.Join(dir, base+ext)
                if _, err := os.Stat(outPath); err == nil {
                    outBytes, _ := ioutil.ReadFile(outPath)
                    expect[ext[1:]] = string(outBytes)
                }
            }
            cases = append(cases, GoldenCase{
                Name:   base,
                Source: string(srcBytes),
                Expect: expect,
            })
        }
    }
    return cases, nil
}

// Diff outputs for golden test, returns empty string if equal.
func Diff(expected, actual string) string {
    if expected == actual {
        return ""
    }
    // Simple line-by-line diff
    expLines := strings.Split(expected, "\n")
    actLines := strings.Split(actual, "\n")
    var buf bytes.Buffer
    for i := 0; i < len(expLines) || i < len(actLines); i++ {
        var exp, act string
        if i < len(expLines) {
            exp = expLines[i]
        }
        if i < len(actLines) {
            act = actLines[i]
        }
        if exp != act {
            fmt.Fprintf(&buf, "- %s\n+ %s\n", exp, act)
        }
    }
    return buf.String()
}
