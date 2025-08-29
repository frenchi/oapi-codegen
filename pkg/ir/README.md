# Intermediate Representation (IR)

This package defines a parser-agnostic data model used by the generator.

Goals:
- Decouple code generation from any specific OpenAPI library.
- Enable a single backend for OpenAPI 3.0 and 3.1.
- Preserve backwards compatibility by mapping both versions into the same IR.

Status: scaffold. Expect iterative additions as the adapter and generator evolve.

