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
| Dict get/set                  | (pending)      |
| Dict merge                    | (pending)      |
| Data group_by                 | (pending)      |
| HTTP /echo throughput         | (pending)      |

Run `go test -bench . ./...` to update results.
