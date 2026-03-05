# Contributing

Thanks for your interest in contributing to AdGuard Log Viewer.

## Getting Started

1. Fork the repository and clone your fork
2. Install Go 1.25+
3. Run the tests to make sure everything works:
   ```sh
   make test
   ```

## Development

The project is a single Go module with no external runtime dependencies. The only test dependency is `pgregory.net/rapid` for property-based testing.

Key files:

| File | Purpose |
|------|---------|
| `config.go` | Config file parsing |
| `model.go` | LogEntry and raw JSON structs |
| `parser.go` | NDJSON stream parser |
| `filter.go` | Filter params and composite filter builder |
| `paginator.go` | Reverse ordering and pagination |
| `render.go` | Template data struct and rendering |
| `handler.go` | HTTP handler wiring |
| `main.go` | Entry point |
| `template.html` | Go html/template for the UI |

## Making Changes

1. Create a branch for your change
2. Write or update tests for any new behavior
3. Run the full test suite:
   ```sh
   make test
   ```
4. Verify the MIPS cross-compilation still works:
   ```sh
   make build-mips
   ```
5. Submit a pull request with a clear description of the change

## Testing

The project uses two kinds of tests:

- Unit tests (`*_test.go`) for specific examples and edge cases
- Property-based tests (`*_prop_test.go`) using `pgregory.net/rapid` for correctness properties

Each property test maps to a formal correctness property documented in the design spec. If you change filter logic, pagination, or rendering, make sure the corresponding property tests still pass.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep the binary small — no external dependencies at runtime
- Keep HTML output minimal — the target device has 128MB RAM
- Prefer streaming over loading entire files into memory

## Reporting Issues

Open an issue with:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Router model and firmware version (if relevant)
