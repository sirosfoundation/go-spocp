# File I/O and Serialization Example

This example demonstrates how to load rulesets from files and serialize them efficiently.

## Running the Example

```bash
cd examples/fileio
go run main.go
```

## What It Demonstrates

### 1. Canonical Format Save/Load
- Saving rules to a text file in canonical S-expression format
- Loading rules back from the file
- Displaying the file contents

### 2. Binary Serialization
- Comparing file sizes between text and binary formats
- Demonstrating compression ratios for large rulesets

### 3. Loading with Comments
- Using comments in policy files (`#`, `//`, `;`)
- Automatic comment filtering during load
- Blank line handling

### 4. Direct Engine Loading
- Loading rules directly into an engine
- Saving and restoring engine state
- Testing queries on loaded rules

### 5. Performance Comparison
- Loading performance for large rulesets
- File size comparison
- Format recommendations

## Key Takeaways

- **Text format**: Best for version control and human editing
- **Binary format**: Best for production deployments with large rulesets
- **Comments**: Use liberally to document your policies
- **Direct loading**: Simplifies application code

## See Also

- [FILE_LOADING.md](../../docs/FILE_LOADING.md) - Complete file loading documentation
- [../basic/](../basic/) - Basic SPOCP usage examples
- [../adaptive/](../adaptive/) - Adaptive engine examples
