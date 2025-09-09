package io

import (
    "encoding/csv"
    "os"
)

// LoadCSV loads a CSV file into a slice of maps.
func LoadCSV(path string) ([]map[string]string, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    r := csv.NewReader(f)
    records, err := r.ReadAll()
    if err != nil {
        return nil, err
    }
    if len(records) < 1 {
        return nil, nil
    }
    header := records[0]
    out := make([]map[string]string, 0, len(records)-1)
    for _, row := range records[1:] {
        m := map[string]string{}
        for i, v := range row {
            m[header[i]] = v
        }
        out = append(out, m)
    }
    return out, nil
}

// DumpCSV writes a slice of maps to a CSV file.
func DumpCSV(path string, rows []map[string]string) error {
    if len(rows) == 0 {
        return nil
    }
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()
    w := csv.NewWriter(f)
    // Write header
    var header []string
    for k := range rows[0] {
        header = append(header, k)
    }
    if err := w.Write(header); err != nil {
        return err
    }
    // Write rows
    for _, row := range rows {
        var record []string
        for _, k := range header {
            record = append(record, row[k])
        }
        if err := w.Write(record); err != nil {
            return err
        }
    }
    w.Flush()
    return w.Error()
}
