# Package integration

## Decision

We use interfaces between packages that depend on each other, lite databases, api.
A compile-time checks is included to ensure concrete types implement interfaces.

## Rationale

* Enables mocking for isolated unit tests.
* Allows implementations to change without affecting dependents.
