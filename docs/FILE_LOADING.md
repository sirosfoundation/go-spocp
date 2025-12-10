# File Loading and Serialization

This document describes how to load rulesets from files and serialize them efficiently in the SPOCP engine.

## Overview

The `pkg/persist` package provides:

1. **File Loading**: Read rules from text files (canonical or advanced form)
2. **Binary Serialization**: Efficient binary format for large rulesets
3. **Engine Integration**: Direct loading/saving methods on `Engine` and `AdaptiveEngine`

## Quick Start

### Loading Rules from a File

```go
engine := spocp.NewEngine()

// Load rules from a file (auto-detects format)
if err := engine.LoadRulesFromFile("policies.txt"); err != nil {
    log.Fatal(err)
}
```

### Saving Rules to a File

```go
// Save in canonical format
engine.SaveRulesToFile("policies.txt", persist.FormatCanonical)

// Save in binary format
engine.SaveRulesToFile("policies.spocp", persist.FormatBinary)
```

## File Formats

### Canonical Format (Default)

Text file with one rule per line in canonical S-expression format:

```
(4:http3:GET)
(4:http4:POST)
(4:file11:/etc/passwd)
```

**Advantages:**
- Human-readable (with practice)
- Version control friendly
- Compact for simple rules
- Standard S-expression format

**Disadvantages:**
- Harder to edit manually than advanced form
- Parsing overhead on load

### Advanced Format

Human-readable format (not yet fully implemented in parser, but supported for saving):

```
(http GET)
(http POST)
(file /etc/passwd)
```

**Advantages:**
- Easy to read and write
- Good for documentation

**Disadvantages:**
- Requires conversion to canonical form
- Not as compact

### Binary Format

Efficient binary encoding for large rulesets:

```
File structure:
- Magic: "SPOCP" (5 bytes)
- Version: 1 (1 byte)
- Rule count: N (4 bytes)
- For each rule:
  - Length: L (4 bytes)
  - Data: canonical form (L bytes)
```

**Advantages:**
- Faster loading (no parsing overhead)
- Good for large rulesets
- Versioned format

**Disadvantages:**
- Not human-readable
- Not version control friendly
- May be larger than text for simple rules

## API Reference

### Package: persist

#### LoadFile

```go
func LoadFile(filename string, opts LoadOptions) ([]sexp.Element, error)
```

Loads rules from a file with options:

```go
opts := persist.LoadOptions{
    Format:      persist.FormatCanonical,  // or FormatBinary
    SkipInvalid: false,                    // Continue on parse errors?
    MaxRules:    0,                        // Limit (0 = unlimited)
    Comments:    []string{"#", "//", ";"}, // Comment prefixes
}

rules, err := persist.LoadFile("rules.txt", opts)
```

#### SaveFile

```go
func SaveFile(filename string, rules []sexp.Element, format FileFormat) error
```

Saves rules to a file in the specified format:

```go
// Canonical format
persist.SaveFile("rules.txt", rules, persist.FormatCanonical)

// Binary format
persist.SaveFile("rules.spocp", rules, persist.FormatBinary)
```

#### LoadFileToSlice (Convenience)

```go
func LoadFileToSlice(filename string) ([]sexp.Element, error)
```

Simplified loading with default options:

```go
rules, err := persist.LoadFileToSlice("rules.txt")
```

### Engine Methods

#### LoadRulesFromFile

```go
func (e *Engine) LoadRulesFromFile(filename string) error
```

Load rules directly into the engine:

```go
engine := spocp.NewEngine()
err := engine.LoadRulesFromFile("policies.txt")
```

#### LoadRulesFromFileWithOptions

```go
func (e *Engine) LoadRulesFromFileWithOptions(filename string, opts persist.LoadOptions) error
```

Load with custom options:

```go
opts := persist.LoadOptions{
    SkipInvalid: true,  // Skip malformed rules
    MaxRules:    1000,  // Load at most 1000 rules
}
err := engine.LoadRulesFromFileWithOptions("policies.txt", opts)
```

#### SaveRulesToFile

```go
func (e *Engine) SaveRulesToFile(filename string, format persist.FileFormat) error
```

Save all engine rules to a file:

```go
// Text format
engine.SaveRulesToFile("backup.txt", persist.FormatCanonical)

// Binary format
engine.SaveRulesToFile("backup.spocp", persist.FormatBinary)
```

#### ExportRules / ImportRules

```go
func (e *Engine) ExportRules() []sexp.Element
func (e *Engine) ImportRules(rules []sexp.Element)
```

For programmatic transfer:

```go
// Export from one engine
rules := engine1.ExportRules()

// Import to another
engine2 := spocp.NewEngine()
engine2.ImportRules(rules)
```

## Usage Examples

### Loading with Comments

Create a file `policies.txt`:

```
# HTTP access control
# Updated: 2025-12-10

(4:http3:GET)   # Allow GET requests
(4:http4:POST)  # Allow POST requests

// File access rules
(4:file11:/etc/passwd)
(4:file8:/var/log)
```

Load it:

```go
engine := spocp.NewEngine()
engine.LoadRulesFromFile("policies.txt")  // Comments automatically filtered
```

### Handling Invalid Rules

```go
opts := persist.LoadOptions{
    SkipInvalid: true,  // Don't fail on invalid rules
}

engine := spocp.NewEngine()
err := engine.LoadRulesFromFileWithOptions("untrusted.txt", opts)
// Invalid rules are skipped, valid ones loaded
```

### Large Ruleset Optimization

For large rulesets (>10,000 rules), use binary format:

```go
// Initial save (from canonical)
engine := spocp.NewEngine()
engine.LoadRulesFromFile("large_policy.txt")
engine.SaveRulesToFile("large_policy.spocp", persist.FormatBinary)

// Fast subsequent loads
engine2 := spocp.NewEngine()
opts := persist.LoadOptions{Format: persist.FormatBinary}
engine2.LoadRulesFromFileWithOptions("large_policy.spocp", opts)
```

### Adaptive Engine Example

```go
engine := spocp.NewAdaptiveEngine()

// Load large ruleset - indexing automatically adapts
engine.LoadRulesFromFile("policies.txt")

// Check adaptive decision
stats := engine.Stats()
fmt.Printf("Indexing enabled: %v\n", stats.IndexingEnabled)
fmt.Printf("Unique tags: %d\n", stats.UniqueTags)
fmt.Printf("Avg fanout: %.2f\n", stats.AvgTagFanout)
```

## Performance Considerations

### Format Comparison

| Format     | Load Speed | File Size | Use Case                          |
|------------|------------|-----------|-----------------------------------|
| Canonical  | Medium     | Small     | Default, version control          |
| Advanced   | Slow       | Medium    | Human editing                     |
| Binary     | Fast       | Varies    | Large rulesets, production deploy |

### Binary Format Performance

The binary format has overhead (10 bytes per file + 4 bytes per rule), so:

- **Small rulesets (<100 rules)**: Text format is comparable or better
- **Medium rulesets (100-1,000 rules)**: Binary may be 10-20% larger
- **Large rulesets (>10,000 rules)**: Binary saves ~10-30% space and loads faster

### Load Options Impact

```go
// Fastest loading (skip validation)
opts := persist.LoadOptions{
    SkipInvalid: true,  // Don't validate each rule deeply
}

// Limited loading (for testing)
opts := persist.LoadOptions{
    MaxRules: 100,  // Load first 100 rules only
}
```

## Best Practices

### 1. Use Canonical Format for Version Control

```go
// In development
engine.SaveRulesToFile("policies.txt", persist.FormatCanonical)
// Commit policies.txt to git
```

### 2. Use Binary Format for Production

```go
// Build step: convert to binary
engine.LoadRulesFromFile("policies.txt")
engine.SaveRulesToFile("policies.spocp", persist.FormatBinary)

// Production: load binary
prodEngine := spocp.NewEngine()
prodEngine.LoadRulesFromFile("policies.spocp")
```

### 3. Handle Loading Errors Gracefully

```go
if err := engine.LoadRulesFromFile(filename); err != nil {
    log.Printf("Failed to load rules from %s: %v", filename, err)
    
    // Fall back to default policy
    engine.AddRule("(5:admin)")  // Default: only admin access
}
```

### 4. Validate After Loading

```go
engine.LoadRulesFromFile("policies.txt")

// Verify rule count
if engine.RuleCount() == 0 {
    log.Fatal("No rules loaded!")
}

// Test a known query
allowed, _ := engine.Query("(4:http3:GET)")
if !allowed {
    log.Fatal("Expected policy doesn't work!")
}
```

### 5. Use Comments Liberally

```
# Section: HTTP Access Control
# Purpose: Allow read-only HTTP operations
# Owner: security-team@example.com
# Last updated: 2025-12-10

(4:http3:GET)
(4:http4:HEAD)
```

## File Organization

### Single File

Simple approach for small deployments:

```
policies.txt
```

### Multi-File

For larger systems, organize by domain:

```
policies/
  http.txt       # HTTP rules
  file.txt       # File access rules
  admin.txt      # Admin rules
```

Load all:

```go
files := []string{
    "policies/http.txt",
    "policies/file.txt",
    "policies/admin.txt",
}

engine := spocp.NewEngine()
for _, file := range files {
    if err := engine.LoadRulesFromFile(file); err != nil {
        log.Printf("Warning: failed to load %s: %v", file, err)
    }
}
```

### Binary Cache Pattern

```
policies/
  src/           # Source files (version controlled)
    http.txt
    file.txt
  cache/         # Binary cache (not version controlled)
    http.spocp
    file.spocp
```

Build script:

```go
func buildCache() {
    srcFiles, _ := filepath.Glob("policies/src/*.txt")
    for _, src := range srcFiles {
        base := filepath.Base(src)
        cache := strings.TrimSuffix(base, ".txt") + ".spocp"
        
        engine := spocp.NewEngine()
        engine.LoadRulesFromFile(src)
        engine.SaveRulesToFile("policies/cache/"+cache, persist.FormatBinary)
    }
}
```

## Error Handling

```go
if err := engine.LoadRulesFromFile(filename); err != nil {
    if os.IsNotExist(err) {
        // File doesn't exist
        log.Fatal("Policy file not found")
    } else if strings.Contains(err.Error(), "failed to parse") {
        // Malformed rule
        log.Fatal("Invalid rule syntax")
    } else {
        // Other error
        log.Fatal(err)
    }
}
```

## See Also

- [API.md](../API.md) - Complete API reference
- [README.md](../README.md) - Getting started guide
- [examples/fileio/](../examples/fileio/) - Working examples
