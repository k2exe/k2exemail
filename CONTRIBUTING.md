# Contributing to K2EXEmail

K2EXEmail is developed with reliability, Winlink interoperability, offline operation, and cross-platform behavior as primary requirements.

## Pull Request Gate

Before merging a pull request:

1. Inspect the relevant current source and interfaces before making substantial changes.
2. Keep the change focused and avoid unrelated refactors.
3. Add or update tests for changed behavior when practical.
4. Run:
   - `go test ./...`
   - `go test -race ./...`
   - `go vet ./...`
   - `git diff --check`
5. Inspect the final diff.
6. Require CI to pass.
7. Review all Gitar findings.

Every Gitar finding must be one of:

- fixed;
- explicitly dismissed as a false positive; or
- intentionally accepted with documented rationale.

Gitar is an additional review gate. It does not replace deterministic tests, source inspection, or human review.

Do not automatically apply Gitar-generated code without reviewing and testing the change.

## Project Priorities

When several implementations are possible, prefer:

1. reliability;
2. Winlink compatibility;
3. offline operation;
4. cross-platform support;
5. maintainability;
6. testability;
7. user experience;
8. resource efficiency;
9. dependency cost.
