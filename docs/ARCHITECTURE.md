# Architecture

## Structure

- `cmd/` would be the composition root, handling command-line and signal setup, coordinating the public packages. Elk is currently a library and ships example binaries under `examples/` instead.
- `pkg/` provide focused APIs and reusable behavior. Package constructors may supply convenient defaults for their own implementation details, reading system locations only through `pkg/constants`.
- `internal/` are low-level implementations. They rely on dependency injection from `pkg/`.

## Layering

Configuration and filesystem access flow from the outside in:

1. `pkg/constants` resolves system paths, honoring development environment overrides.
2. `pkg/config` loads `rhsm.conf` and the native configuration, and returns the resolved configuration.
3. `internal/config` merges the embedded defaults, the translated RHSM fallback, the main file, and sorted drop-ins, then resolves and validates the result.
4. `internal/config/rhsm` parses `rhsm.conf` on its own, independent of the native configuration format.

Certificate material follows the same split: `pkg/consumer`, `pkg/product`, and `pkg/certstore` expose the RHSM certificate locations, backed by `internal/rhsmx509` and `internal/certstore`.
