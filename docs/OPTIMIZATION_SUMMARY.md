# Performance Optimization Summary

## Current Implementation Status

✅ **Tag-Based Indexing**: Fully implemented and production-ready
- Available in both `Engine` (always-on) and `AdaptiveEngine` (automatic)
- 2-5x speedup for typical workloads
- Zero allocations during queries
- Minimal memory overhead (~24 bytes per rule)

✅ **Adaptive Strategy**: Automatically optimizes based on ruleset characteristics
- Thresholds: ≥50 rules, ≥5 unique tags, ≤100 avg fanout
- No configuration required
- See [ADAPTIVE_ENGINE.md](ADAPTIVE_ENGINE.md)

✅ **File Loading & Serialization**: Efficient bulk loading
- Text format: Version control friendly
- Binary format: Fast loading for large rulesets
- See [FILE_LOADING.md](FILE_LOADING.md)

## Performance Characteristics

| Ruleset Size | Indexed Query Time | Non-Indexed Query Time | Speedup |
|--------------|-------------------|------------------------|---------|
| 100 rules    | 735 ns            | 2,338 ns               | 3.2x    |
| 1,000 rules  | 7,364 ns          | 29,047 ns              | 3.9x    |
| 10,000 rules | 95,566 ns         | 279,023 ns             | 2.9x    |
| 50,000 rules | 1,055,105 ns      | 1,913,802 ns           | 1.8x    |

See [PERFORMANCE_REPORT.md](../PERFORMANCE_REPORT.md) for complete benchmarks.

## How to Use

### Recommended: Use AdaptiveEngine

```go
engine := spocp.NewAdaptiveEngine()
// or
engine := spocp.New()

// Load rules from file
engine.LoadRulesFromFile("policies.txt")

// Indexing automatically enabled when beneficial
stats := engine.Stats()
fmt.Printf("Indexing enabled: %v\n", stats.IndexingEnabled)
```

### Alternative: Manual Control

```go
// Always indexed (default)
engine := spocp.NewEngine()

// Disable indexing
engine := spocp.NewEngineWithIndexing(false)
```

## When to Use Each Engine

**Use AdaptiveEngine when:**
- You want automatic optimization ✅ (most cases)
- Ruleset size varies over time
- You're unsure about indexing benefits

**Use Engine (always-indexed) when:**
- You know you have >1000 rules
- You want predictable performance
- You're benchmarking

**Use NewEngineWithIndexing(false) when:**
- Rules < 100 (minimal benefit)
- Memory is extremely constrained
- Benchmarking without indexing

## Future Optimization Opportunities

If you need even better performance (rare), consider:

1. **Query Result Caching**: Near-instant for repeated queries
   - Memory cost: ~100 bytes per cached query
   - Best for: Repetitive authorization checks

2. **Parallel Evaluation**: 2-4x speedup on multi-core systems
   - Best for: Very large rulesets (>10,000 rules)
   - Only helps with complex rule matching

3. **Trie-Based Indexing**: 100-1000x for hierarchical paths
   - Best for: File paths, URLs, tree structures
   - Memory cost: ~80 bytes per trie node

4. **Compiled Bytecode**: 30-50% improvement
   - High implementation complexity
   - Only for extreme performance requirements

See [PERFORMANCE_VISUAL.md](PERFORMANCE_VISUAL.md) for visual comparisons.

## Performance Measurement

Use the built-in benchmarks to measure performance:

```bash
# Run all benchmarks
make bench

# Long benchmark suite (more accurate)
make bench-long

# Compare changes
make bench-long > before.txt
# ... make changes ...
make bench-long > after.txt
benchstat before.txt after.txt
```

## Interpreting Results

Current indexed performance with AdaptiveEngine:

- ✅ <1,000 rules: **Excellent** (735-7,364 ns per query)
- ✅ 1,000-10,000 rules: **Good** (7-95 µs per query)
- ✅ 10,000-50,000 rules: **Acceptable** (95-1,055 µs per query)

Without indexing, larger rulesets degrade faster:

- ⚠️ 10,000 rules: ~279 µs per query (3x slower)
- ❌ 50,000 rules: ~1,913 µs per query (2x slower)

**When the current implementation is sufficient:**
- Queries complete in acceptable time for your use case
- Memory usage is reasonable
- AdaptiveEngine provides automatic optimization

**When additional optimization may be needed:**
- Queries must complete in <1µs (consider caching)
- Processing >100,000 queries/second
- Working with >100,000 rules

See [ADAPTIVE_ENGINE.md](ADAPTIVE_ENGINE.md) for engine selection guidance.
