# Engine Selection Guide

This document helps you choose the right engine for your use case.

## Quick Decision

**✅ Use `spocp.New()` for 95% of use cases**

This returns an adaptive engine that automatically optimizes. Only use the regular `Engine` if you fall into one of the advanced categories below.

## Decision Tree

```
Do you need to explicitly control indexing?
│
├─ NO → Use spocp.New() ✅
│       (Recommended for production)
│
└─ YES → Why do you need explicit control?
          │
          ├─ Benchmarking/comparing strategies
          │  → Use NewEngine() and NewEngineWithIndexing(false)
          │
          ├─ Testing indexing behavior
          │  → Use NewEngineWithIndexing(true/false)
          │
          ├─ Profiled and know optimal strategy
          │  → Use NewEngine() or NewEngineWithIndexing(false)
          │
          └─ Adaptive overhead matters (<0.1%)
             → Use NewEngine()
```

## Detailed Comparison

| Feature | AdaptiveEngine | Engine (Regular) |
|---------|---------------|------------------|
| **Automatic Optimization** | ✅ Yes | ❌ Manual |
| **Configuration Required** | ❌ None | ✅ Choose indexing |
| **Best For** | Production use | Benchmarking/testing |
| **Performance** | Optimal | Depends on choice |
| **Overhead** | ~0.1% (statistics) | None |
| **Simplicity** | High | Medium |
| **Recommended** | ✅ Yes | Only for advanced use |

## Use Case Examples

### ✅ Use AdaptiveEngine When:

**Web API Authorization**
```go
// API with varying number of endpoints
engine := spocp.New()
// Automatically adapts as endpoints are added
```

**Multi-Tenant Applications**
```go
// Different tenants have different rule counts
func NewTenantEngine() *spocp.AdaptiveEngine {
    return spocp.New()
    // Each tenant gets optimal strategy
}
```

**Dynamic Rule Loading**
```go
// Rules loaded from database/config files
engine := spocp.New()
for _, rule := range loadedRules {
    engine.AddRule(rule)
}
// Adapts based on actual ruleset
```

**General Purpose Authorization**
```go
// You don't know ruleset size in advance
engine := spocp.New()
// Always optimal performance
```

### ⚠️ Use Regular Engine When:

**Performance Benchmarking**
```go
func BenchmarkIndexedVsLinear(b *testing.B) {
    indexed := spocp.NewEngine()  // Always indexed
    linear := spocp.NewEngineWithIndexing(false)  // Never indexed
    
    // Compare performance...
}
```

**Testing Indexing Behavior**
```go
func TestIndexingLogic(t *testing.T) {
    // Test that indexing works correctly
    indexed := spocp.NewEngineWithIndexing(true)
    
    // Test that linear search works correctly
    linear := spocp.NewEngineWithIndexing(false)
}
```

**Known Optimal Strategy**
```go
// After profiling, you know your workload benefits from indexing
// AND ruleset characteristics never change
func NewProductionEngine() *spocp.Engine {
    return spocp.NewEngine()  // Always indexed
}
```

**Extremely Tight Performance Requirements**
```go
// The ~0.1% overhead of adaptive statistics matters
// (very rare - measure first!)
func NewHighPerfEngine() *spocp.Engine {
    return spocp.NewEngine()
}
```

## Performance Characteristics

### AdaptiveEngine

| Scenario | Behavior | Performance |
|----------|----------|-------------|
| 10 rules | No indexing | ~0.1 µs/query |
| 100 rules, 2 tags | No indexing | ~2 µs/query |
| 100 rules, 10 tags | **Indexing enabled** | ~0.5 µs/query (5x faster) |
| 1000 rules, 50 tags | **Indexing enabled** | ~0.5 µs/query (100x faster) |

**Overhead**: ~0.1% from statistics tracking (negligible)

### Regular Engine (Indexed)

| Scenario | Performance | Notes |
|----------|-------------|-------|
| 10 rules | ~0.15 µs/query | Slight overhead |
| 100 rules, 2 tags | ~2 µs/query | Not optimal (few tags) |
| 100 rules, 10 tags | ~0.5 µs/query | Optimal |
| 1000 rules, 50 tags | ~0.5 µs/query | Optimal |

**Note**: Always has indexing overhead, even when not beneficial

### Regular Engine (Non-Indexed)

| Scenario | Performance | Notes |
|----------|-------------|-------|
| 10 rules | ~0.1 µs/query | Optimal for small sets |
| 100 rules | ~2 µs/query | OK |
| 1000 rules | ~20 µs/query | Slow (40x slower than indexed) |

**Note**: Linear scaling - not suitable for large rulesets

## Migration Guide

### From Regular Engine to Adaptive

**Before:**
```go
// Manual decision required
engine := spocp.NewEngine()  // Is this right for my use case?
```

**After:**
```go
// Automatic optimization - shortest and cleanest!
engine := spocp.New()  // Always right!

// Or use the explicit name if you prefer:
engine := spocp.NewAdaptiveEngine()  // Same thing
```

**Compatibility**: `AdaptiveEngine` implements the same interface, so no other changes needed.

### Keeping Regular Engine for Tests

If you have existing code using `NewEngine()` for actual application logic (not tests), you can:

1. **Migrate gradually**: Replace with `NewAdaptiveEngine()` one component at a time
2. **Keep for tests**: Tests using `NewEngineWithIndexing(false)` should stay as-is
3. **Benchmarks unchanged**: Benchmarking code needs explicit control

## API Compatibility

Both engines implement the same core interface:

```go
type AuthEngine interface {
    AddRule(rule string) error
    AddRuleElement(rule sexp.Element)
    Query(query string) (bool, error)
    QueryElement(query sexp.Element) bool
    FindMatchingRules(query string) ([]sexp.Element, error)
    RuleCount() int
    Clear()
}
```

You can swap between them without code changes:

```go
var engine AuthEngine

// For production
engine = spocp.NewAdaptiveEngine()

// For testing/benchmarking
engine = spocp.NewEngine()
```

## When NOT to Overthink

**Just use `NewAdaptiveEngine()`!**

Unless you're:
- Writing benchmarks
- Testing indexing logic
- Have profiled and proven the 0.1% overhead matters (it doesn't)

The adaptive engine handles everything automatically and performs optimally in all scenarios.

## Summary

| Your Situation | Recommended Engine |
|----------------|-------------------|
| **Production code** | `spocp.New()` ✅ |
| **Unsure which to use** | `spocp.New()` ✅ |
| **Writing benchmarks** | `NewEngine()` / `NewEngineWithIndexing()` |
| **Testing indexing** | `NewEngineWithIndexing(true/false)` |
| **Profiled a specific need** | Consider regular `Engine` (rare) |

**Default recommendation**: Use `spocp.New()` and only switch if you have a specific, measured reason.

**Note**: `spocp.New()` is an alias for `spocp.NewAdaptiveEngine()` - both return the same type.
