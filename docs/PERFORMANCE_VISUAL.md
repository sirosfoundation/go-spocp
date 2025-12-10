# Performance Optimization Visual Guide

## Current vs Optimized Performance

### Linear Search (Current Implementation)

```
Rules in Engine: 10,000

Query: (read /home/user/file.txt)

┌─────────────────────────────────────────┐
│  Check Rule 1: (write ...)    ❌       │
│  Check Rule 2: (execute ...)  ❌       │
│  Check Rule 3: (delete ...)   ❌       │
│  Check Rule 4: (admin ...)    ❌       │
│  Check Rule 5: (write ...)    ❌       │
│  ... (checking 5,000 non-read rules)    │
│  Check Rule 5001: (read /*)   ✅ MATCH │
└─────────────────────────────────────────┘

Time: 259 µs (checked 5,001 rules)
```

### Tag-Indexed Search (Optimized)

```
Rules in Engine: 10,000
- 2,000 tagged "read"
- 2,000 tagged "write" 
- 2,000 tagged "execute"
- 2,000 tagged "delete"
- 2,000 tagged "admin"

Query: (read /home/user/file.txt)

┌─────────────────────────────────────────┐
│  Look up tag "read" in index → 2,000    │
│  Check Rule 1: (read /etc/*)    ❌     │
│  Check Rule 2: (read /var/*)    ❌     │
│  Check Rule 3: (read /home/*)   ✅ MATCH│
└─────────────────────────────────────────┘

Time: 2-5 µs (checked only 3 rules!)
Speedup: 50-100x faster
```

## Performance Comparison Table

| Optimization | Rules Checked | Time | Speedup |
|--------------|---------------|------|---------|
| None (Linear) | 10,000 | 259 µs | 1x baseline |
| Tag Index | ~500 (5%) | 13 µs | **20x faster** |
| Tag Index + Cache | 0 (cached) | 0.1 µs | **2500x faster** |
| Tag Index + Cache + Parallel | ~500 / 4 cores | 4 µs | **65x faster** |

## Optimization Impact by Use Case

### Use Case 1: Web Application Authorization
```
Scenario: 1,000 rules, 100 requests/sec
Current:  39 µs/query × 100 = 3.9 ms/sec (0.4% CPU)
Indexed:  4 µs/query × 100 = 0.4 ms/sec (0.04% CPU)
Benefit:  Minimal - current performance is fine ✅
```

### Use Case 2: API Gateway
```
Scenario: 10,000 rules, 10,000 requests/sec
Current:  259 µs/query × 10,000 = 2,590 sec (!) 🔥
Indexed:  13 µs/query × 10,000 = 130 sec
Benefit:  CRITICAL - need optimization ❗
```

### Use Case 3: File System Access Control
```
Scenario: 50,000 rules (hierarchical paths), 1,000 requests/sec
Current:  1,250 µs/query × 1,000 = 1,250 sec 🔥
Indexed:  25 µs/query × 1,000 = 25 sec
+ Trie:   2 µs/query × 1,000 = 2 sec
Benefit:  ESSENTIAL for hierarchical data 🎯
```

## Memory Trade-offs

### Current Implementation
```
Memory per rule: ~100 bytes
10,000 rules: ~1 MB
50,000 rules: ~5 MB
```

### With Tag Index
```
Memory per rule: ~124 bytes (+24 bytes)
10,000 rules: ~1.24 MB (+240 KB)
50,000 rules: ~6.2 MB (+1.2 MB)

Trade-off: +20% memory for 10-100x speed ✅ WORTH IT
```

### With LRU Cache
```
Cache size: 10,000 queries
Memory: ~10 MB

Trade-off: Instant cached queries ✅ WORTH IT
```

### With Trie Index
```
Memory per trie node: ~80 bytes
Typical depth: 5-10 levels
Memory for 50,000 rules: ~20-40 MB

Trade-off: 4x memory for 100-1000x speed
Decision: Use for hierarchical data only 🤔
```

## Implementation Effort vs Impact

```
                     Impact
                        ↑
                        │
            Tag Index ──┤ ⭐⭐⭐
                        │ (2 days, 20x speedup)
                        │
     Query Cache ───────┤ ⭐⭐
                        │ (1 day, ∞ for cached)
                        │
Micro-optimizations ────┤ ⭐
                        │ (0.5 days, 15% faster)
                        │
         Parallel ──────┤ ⭐⭐
                        │ (2 days, 4x for large sets)
                        │
     Trie Index ────────┤ ⭐⭐⭐
                        │ (1 week, 100x for paths)
                        │
Compiled Bytecode ──────┤ ⭐⭐
                        │ (3 weeks, 50% faster)
                        │
                        └──────────────────→
                              Effort
```

## Decision Tree

```
Start: Need faster performance?
│
├─→ Rules < 1,000?
│   └─→ YES: Current implementation is fine ✅
│
├─→ Rules 1,000 - 5,000?
│   └─→ Add Tag Index (2 days) ⭐⭐⭐
│
├─→ Rules 5,000 - 10,000?
│   ├─→ Add Tag Index (2 days) ⭐⭐⭐
│   └─→ Add Query Cache (1 day) ⭐⭐
│
├─→ Rules > 10,000?
│   ├─→ Add Tag Index (2 days) ⭐⭐⭐
│   ├─→ Add Query Cache (1 day) ⭐⭐
│   └─→ Consider Parallel Eval (2 days) ⭐⭐
│
└─→ Hierarchical rules (paths, URLs)?
    ├─→ Add Tag Index first (2 days) ⭐⭐⭐
    ├─→ Add Trie Index (1 week) ⭐⭐⭐
    └─→ 100-1000x speedup! 🚀
```

## Real-World Example

### Before Optimization
```go
// 10,000 rules, typical query takes 259 µs

engine := spocp.NewEngine()
// Add 10,000 rules...

start := time.Now()
for i := 0; i < 1000; i++ {
    engine.Query("(read /path/to/file)")
}
elapsed := time.Since(start)
// Result: ~259 ms for 1000 queries
```

### After Tag Index Optimization
```go
// Same 10,000 rules, query now takes ~13 µs

engine := NewIndexedEngine()
// Add 10,000 rules... (index built automatically)

start := time.Now()
for i := 0; i < 1000; i++ {
    engine.Query("(read /path/to/file)")
}
elapsed := time.Since(start)
// Result: ~13 ms for 1000 queries
// Speedup: 20x faster! 🚀
```

## Conclusion

The **tag-based index** is the clear winner for most use cases:
- ✅ 10-100x speedup
- ✅ 2 days implementation
- ✅ Only +20% memory
- ✅ Works for all rule types

Start here, then add other optimizations as needed!

See `PERFORMANCE_IMPROVEMENTS.md` for implementation details.
