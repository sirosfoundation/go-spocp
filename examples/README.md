# SPOCP Examples

This directory contains runnable examples demonstrating various features of the SPOCP library.

## Available Examples

### basic
Basic usage example showing:
- Creating an engine with `spocp.New()`
- Adding rules
- Querying policies
- Using wildcards and sets

Run with:
```bash
cd examples/basic
go run main.go
```

### adaptive
Demonstrates the AdaptiveEngine with automatic indexing:
- Creating rules with diverse tags
- Automatic indexing optimization
- Performance statistics
- Index transition monitoring

Run with:
```bash
cd examples/adaptive
go run main.go
```

## Building Examples

Build all examples:
```bash
for example_dir in examples/*/; do
  (cd "$example_dir" && go build .)
done
```

Build a specific example:
```bash
cd examples/basic
go build .
```

## Example Structure

Each example is in its own subdirectory with a `main.go` file. This structure:
- Allows each example to be built independently
- Prevents conflicts between multiple `main()` functions
- Makes it easy to add new examples
