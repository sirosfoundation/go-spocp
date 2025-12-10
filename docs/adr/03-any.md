# any instead of interface{}

## Decision

We use the newer 'any' instead of 'interface{}' to represent a value that can have any type, even in complex types example: map[string]interface{} becomes map[string]any

## Rationale

- Functionality: Both any and interface{} can hold values of any type.
- Interchangeability: They can be used interchangeably in code.
- Readability: any is generally considered more readable, especially for newcomers to Go.
- Compiler treatment: The compiler treats them identically.