# AGENTS

- Always run `make fmt` and `make check` after making changes on Go code.
- Always prefer fixing a linter issue to ignoring it. Every linter ignore line needs to be escalated to the user.
- You are not allowed to alter `.golangci.yml` unless told explicitly. Fix the linter violations instead.
- Every Go method (public AND private) should have a docstring describing _what_ it does.
- Every Go unit test should additionally have a gherkin statement further explaining the test: "Given {state}, when {action}, then {reaction}."
- Every package should have a `doc.go` file, describing how to use its public API, including a snippet demonstrating it.
- Only the `pkg/constants` package may read environment variables. Other packages receive their locations through parameters or the constants API.
- Packages in `pkg/` must not depend on `cmd/`; packages in `internal/` must not depend on `cmd/` or `pkg/`.
