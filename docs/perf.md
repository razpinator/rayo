# Performance Benchmarks

## Targets
- Parsing large files
- Codegen time for large files
- Dict operations: get, set, merge
- Data helpers: group_by
- HTTP server throughput for /echo

## Current Numbers

| Benchmark                     | Result         |
|------------------------------|----------------|
| Parsing large file            | 1548 ns/op (752413 iterations) |
| Codegen large file            | 68.06 ns/op (17084173 iterations) |
| Dict get/set                  | 35.00 ns/op (33949582 iterations) |
| Dict merge                    | 132341 ns/op (9290 iterations) |
| Data group_by                 | 201325 ns/op (5917 iterations) |
| HTTP /echo throughput         | 126126 ns/op (9766 iterations) |

Run `go test -bench . ./...` to update results.
