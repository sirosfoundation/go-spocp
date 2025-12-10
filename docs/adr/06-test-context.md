# test Context

## Decision

We use test.Context() in tests

## Rationale

* Simple to handle the Context within the test.
* the context is canceled just before Cleanup-registered functions are called.
