# Contributing to Vector

Send issues and pull requests to [CaliLuke/vector](https://github.com/CaliLuke/vector). Use `main` as the base branch.

Read the [development guide](docs/development.md) before changing code. Describe the user-visible problem and the expected behavior.
For bugs, include reproduction steps, the revision, and relevant logs with secrets removed.
Report security vulnerabilities through the process in [SECURITY.md](SECURITY.md).

## Changes and tests

Keep changes focused. Preserve authentication, tenant isolation, data integrity, and durable job behavior.
Add a regression test for a fixed defect. Run affected tests, formatting, lint, and type checks before submitting.
For Go changes, run `go test ./...` and `go vet ./...` from `apps/go-api`.
Database and proxy changes also need the relevant integration tests.

Do not edit generated transport code. Edit its design and regenerate the affected package.
Keep existing copyright notices and third-party attribution. Contributions use the project AGPL-3.0-only license.
Never commit credentials, local environment files, production data, or private review notes.

## Frontend

Shared UI lives in `packages/ui` and `packages/propel`. Application code lives in `apps/web`, `apps/admin`, and `apps/space`.
Internal packages still use the `@plane/*` namespace for source compatibility.
Use existing shared components and helpers. Preserve keyboard navigation, focus behavior, and disabled states.

Translations live in `packages/i18n/src/locales`. Keep translation keys and ICU placeholders consistent across languages.
Preserve identifiers when changing product copy.

## Review

Explain what changed, why it changed, and which checks passed. Include screenshots for visible changes.
