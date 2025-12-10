# Adaptive Engine

The `AdaptiveEngine` automatically decides whether to use tag-based indexing based on the characteristics of your ruleset. This provides optimal performance without requiring manual configuration.

## Overview

The adaptive engine monitors three key metrics as rules are added:
1. **Total rule count** - Need enough rules for indexing overhead to be worthwhile
2. **Tag diversity** - Need enough unique tags for selective lookups
3. **Tag fanout** - Average rules per tag (should be low for selectivity)

## When Indexing is Enabled

Indexing is automatically enabled when **ALL** of these conditions are met:

| Condition | Threshold | Reason |
|-----------|-----------|--------|
| Total Rules | ≥ 50 | Indexing overhead not worth it for small rulesets |
| Unique Tags | ≥ 5 | Need tag diversity for selective lookups |
| Avg Fanout | ≤ 100 | Tags should be selective enough to narrow search |

## Usage

### Basic Usage

```go
import "github.com/sirosfoundation/go-spocp"

// Create an adaptive engine
engine := spocp.NewAdaptiveEngine()

// Add rules - indexing automatically adjusts
for i := 0; i < 100; i++ {
    engine.AddRule("(4:read4:file)")
}

// Query - uses optimal strategy automatically
allowed, err := engine.Query("(4:read4:file)")

// Check if indexing is active
stats := engine.Stats()
fmt.Printf("Indexing enabled: %v\n", stats.IndexingEnabled)
```

### Monitoring Adaptive Behavior

```go
engine := spocp.NewAdaptiveEngine()

// Add rules gradually
for i := 0; i < 200; i++ {
    tags := []string{"read", "write", "delete", "update", "create"}
    tag := tags[i%len(tags)]
    rule := fmt.Sprintf("(%d:%s4:file)", len(tag), tag)
    engine.AddRule(rule)
    
    // Watch indexing decision change
    if i%50 == 0 {
        stats := engine.Stats()
        fmt.Printf("Rules: %d, Indexing: %v\n", 
            stats.TotalRules, stats.IndexingEnabled)
    }
}
```

### Getting Statistics

```go
stats := engine.Stats()

fmt.Printf("Total Rules: %d\n", stats.TotalRules)
fmt.Printf("List Rules: %d\n", stats.ListRules)
fmt.Printf("Atom Rules: %d\n", stats.AtomRules)
fmt.Printf("Unique Tags: %d\n", stats.UniqueTags)
fmt.Printf("Avg Fanout: %.2f\n", stats.AvgTagFanout)
fmt.Printf("Indexing: %v\n", stats.IndexingEnabled)
```

### Manual Override

For testing or special cases, you can override the automatic decision:

```go
engine := spocp.NewAdaptiveEngine()

// Add a few rules (normally wouldn't enable indexing)
for i := 0; i < 10; i++ {
    engine.AddRule("(4:test)")
}

// Force enable indexing for testing
engine.ForceIndexing(true)

// Or force disable
engine.ForceIndexing(false)
```

## Decision Logic

The adaptive engine recalculates its strategy after **every** rule addition:

```go
shouldIndex := 
    totalRules >= 50 &&        // Enough rules to justify overhead
    uniqueTags >= 5 &&         // Enough tag diversity
    avgFanout <= 100           // Tags are selective enough
```

### Example Scenarios

#### ✅ Indexing Enabled

**Scenario**: API gateway with many endpoints
- 200 rules
- 50 unique tags (different endpoints)
- Avg fanout: 4 rules/tag
- **Result**: Indexing enabled - highly selective tags

#### ❌ Indexing Disabled - Small Ruleset

**Scenario**: Simple file permissions
- 20 rules
- 8 unique tags
- **Result**: Indexing disabled - too few rules

#### ❌ Indexing Disabled - Low Diversity

**Scenario**: Binary permissions (read/write only)
- 100 rules
- 2 unique tags
- **Result**: Indexing disabled - not enough tag diversity

#### ❌ Indexing Disabled - High Fanout

**Scenario**: Single catch-all tag
- 500 rules
- 5 unique tags
- Avg fanout: 100 rules/tag
- **Result**: Indexing disabled - tags not selective

## Performance Impact

### Small Rulesets (< 50 rules)
- **Without indexing**: ~0.1 µs per query
- **With indexing**: ~0.15 µs per query (overhead not justified)
- **Adaptive choice**: No indexing ✓

### Large Rulesets with Good Tags (100+ rules, 10+ tags)
- **Without indexing**: ~10 µs per query (linear scan)
- **With indexing**: ~0.5 µs per query (direct lookup)
- **Speedup**: 20x faster ✓
- **Adaptive choice**: Use indexing ✓

### Large Rulesets with Poor Tags (100+ rules, 2 tags)
- **Without indexing**: ~10 µs per query
- **With indexing**: ~8 µs per query (still scans ~50 rules)
- **Speedup**: 1.25x (minimal benefit)
- **Adaptive choice**: No indexing (overhead not justified) ✓

## API Compatibility

The `AdaptiveEngine` implements the same interface as the regular `Engine`:

```go
type Engine interface {
    AddRule(rule string) error
    AddRuleElement(rule sexp.Element)
    Query(query string) (bool, error)
    QueryElement(query sexp.Element) bool
    FindMatchingRules(query string) ([]sexp.Element, error)
    RuleCount() int
    Clear()
    GetIndexStats() map[string]any
}
```

You can swap between `NewEngine()` and `NewAdaptiveEngine()` without code changes:

```go
// Regular engine with manual indexing control
engine := spocp.NewEngine()  // indexing always on

// Adaptive engine with automatic control
engine := spocp.NewAdaptiveEngine()  // indexing adapts

// Use identically
engine.AddRule("(4:read4:file)")
allowed, _ := engine.Query("(4:read4:file)")
```

## Tuning Thresholds

If you need different thresholds for your use case, you can fork the `adaptive_engine.go` and modify these constants:

```go
const (
    minRulesForIndexing     = 50   // Minimum rules to enable indexing
    minTagCountForIndexing  = 5    // Minimum unique tags required
    maxAvgFanoutForIndexing = 100  // Maximum avg rules per tag
)
```

## Best Practices

### ✅ Do Use Adaptive Engine When:
- You don't know the ruleset size in advance
- Rulesets vary significantly between deployments
- You want optimal performance without tuning
- You're prototyping and want simplicity

### ⚠️ Consider Regular Engine When:
- You have deep knowledge of your ruleset characteristics
- Performance requirements are extremely tight
- You want explicit control over indexing
- Profiling shows adaptive overhead matters (rare)

### 🔧 Use ForceIndexing When:
- Testing indexing behavior with small datasets
- Benchmarking and profiling
- Debugging index-related issues
- Temporarily disabling indexing for diagnostics

## Migration Guide

### From Regular Engine to Adaptive Engine

```go
// Before
engine := spocp.NewEngine()  // Always indexed

// After
engine := spocp.NewAdaptiveEngine()  // Auto-adapts

// Everything else stays the same!
```

### From Non-Indexed Engine to Adaptive Engine

```go
// Before
engine := spocp.NewEngineWithIndexing(false)  // Never indexed

// After
engine := spocp.NewAdaptiveEngine()  // Auto-adapts

// If you need to ensure indexing stays off:
engine := spocp.NewAdaptiveEngine()
engine.ForceIndexing(false)
```

## Implementation Details

The adaptive engine:
1. **Always maintains index structures** - no performance penalty when indexing is disabled
2. **Recalculates on every AddRule** - ensures optimal strategy as ruleset grows
3. **Zero query overhead** - decision made once at add time, not query time
4. **Thread-unsafe** - wrap with sync.RWMutex if needed for concurrent access

## Examples

See `examples/adaptive_demo.go` for a complete demonstration of:
- Small ruleset behavior
- Large diverse ruleset behavior
- Poor tag diversity behavior
- Manual override usage
- Statistics monitoring
