# AuthZen to S-Expression Mapping Strategy

## Decision

AuthZen evaluation requests are converted to SPOCP S-expressions using a canonical structure:

```lisp
(resource-type
  (resource-properties...)
  (action action-name (action-properties...))
  (subject (type subject-type) (id subject-id) (subject-properties...))
  (context (context-properties...)))
```

Resource type becomes the root tag, and all components are nested within.

## Rationale

**Consistency**: This structure provides a predictable pattern for writing SPOCP rules that match AuthZen requests.

**Readability**: The hierarchical structure mirrors the AuthZen request format, making rules easier to understand.

**Extensibility**: Properties on any component (subject, resource, action, context) map uniformly through `propertyToSExp()`, supporting arbitrary nested data.

**Type Safety**: Strong typing in Go (`Subject`, `Resource`, `Action` structs) prevents malformed requests.

## Alternatives Considered

**Flat Structure**: `(resource-type resource-id action subject-id ...)`

- Rejected: Doesn't scale to complex properties, loses structure

**Subject as Root**: `(subject-type (resource resource-type ...) (action ...))`

- Rejected: Less intuitive for resource-centric authorization (most common case)

**Direct JSON Embedding**: Store AuthZen JSON in SPOCP atoms

- Rejected: Defeats SPOCP's pattern matching strength

## Consequences

**Positive**:

- Rules can pattern-match on any level of hierarchy
- Properties support full JSON types (strings, numbers, booleans, arrays, objects)
- Clear documentation path for rule authors
- Examples in `examples/rules/authzen.spoc` demonstrate patterns

**Negative**:

- Conversion overhead on every request
- Rule authors must understand both AuthZen and S-expression formats
- Nested properties create deep S-expression trees

**Implementation Notes**:

- `EvaluationRequest.ToSExpression()` handles conversion
- `propertyToSExp()` recursively converts JSON types
- `buildList()` helper works around variadic parameter requirements
- Comprehensive test coverage in `authzen_test.go`
