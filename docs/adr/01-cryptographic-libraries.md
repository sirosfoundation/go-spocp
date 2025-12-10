# Cryptographic libraries

## Decision

This project tries to avoid implementing cryptographic primitives, favouring the reuse of existing, well-tested libraries.

## Rationale

Cryptography is hard to get right and making a mistake when implementing a cryptographic primitive will have serious implications for the security of protocols that build upon those primitives.