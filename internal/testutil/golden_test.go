package testutil

import "testing"

func TestGolden(t *testing.T) {
    cases, err := LoadGoldenCases("../../testdata/golden")
    if err != nil {
        t.Fatalf("failed to load golden cases: %v", err)
    }
    if len(cases) == 0 {
        t.Errorf("no golden cases found")
    }
    // Add more checks as needed
}