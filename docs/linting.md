# Linting in Plane - How It Works

We use [OxLint](https://oxc.rs/docs/guide/usage/linter) for linting across the entire monorepo. OxLint is a single Rust binary that's 50-100x faster than ESLint, with zero Node.js dependencies at runtime.

## Key Points

1. **Single Root Config** - One `.oxlintrc.json` at the repo root handles all packages and apps
2. **No Build Required** - OxLint doesn't need TypeScript build artifacts, so lint runs independently of build
3. **Plugin Coverage** - react, typescript, jsx-a11y, import, promise, unicorn, oxc

## How to Run

From the root of the repo:

```bash
# Check lint diagnostics and TypeScript errors against the repository ratchets
bun run check:lint

# Auto-fix lint errors
bun run fix:lint
```

To run the package-local Oxlint command:

```bash
bun run turbo run check:lint --filter=@plane/ui
```

The repository-wide command is the canonical check. Package-local commands are useful during development; they do not replace its per-finding diagnostic baseline.

## VS Code Integration

Install the [OxLint extension](https://marketplace.visualstudio.com/items?itemName=nicolo-ribaudo.vscode-oxlint) for inline errors/warnings as you type.

## What Gets Linted

The config applies to all TypeScript and JavaScript files across:

- `apps/web`, `apps/admin`, `apps/space`, `apps/live`
- All packages in `packages/`

**Ignored paths:**

- `node_modules/`, `dist/`, `build/`, `.next/`, `.turbo/`
- Config files (`*.config.{js,mjs,cjs,ts}`)
- Public folders, coverage, storybook-static

## Rules Overview

OxLint uses category-based configuration:

| Category        | Level | What It Catches                          |
| --------------- | ----- | ---------------------------------------- |
| **correctness** | error | Real bugs that will cause runtime errors |
| **suspicious**  | warn  | Code patterns that are likely mistakes   |
| **perf**        | warn  | Performance anti-patterns                |

Additional rule overrides:

- `react/prop-types` off (TypeScript handles prop validation)
- `no-unused-vars` warns with `_` prefix pattern ignored
- Several noisy unicorn rules disabled

## Backward Compatibility

OxLint supports `eslint-disable` comments, so existing inline suppressions continue to work.

## Suppressing Warnings

```typescript
// Single line
// eslint-disable-next-line no-unused-vars
const data = response;

// Block
/* eslint-disable no-unused-vars */
// ... code
/* eslint-enable no-unused-vars */
```

**Please use sparingly** - most warnings indicate real issues that should be fixed.

## Pre-commit Hook

Lint-staged runs automatically on commit via Husky:

- oxfmt formats your staged files
- OxLint fixes what it can (with `--deny-warnings`)

If the commit fails due to lint errors, fix them before committing.

## Reference Files

- [.oxlintrc.json](../.oxlintrc.json) - OxLint configuration
- [package.json](../package.json) - Available scripts

## Observability audit

Run this opt-in source audit from the repository root after installing the pinned Bun dependencies.

```bash
source scripts/dev-env.sh
bun run check:observability
bun run check:observability --format json apps/web/core/queries apps/go-api/project_member_preferences.go
bun run check:observability --check apps/go-api/project_member_preferences.go
```

The default command prints review candidates and exits 0 after a successful scan.
Use `--check` to exit 1 when the selected files contain any findings. A parser or tool failure exits 2.
There is no findings baseline. The audit runs independently of the normal lint and CI gates.

| Rule                             | What to review                                                                                                                           |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `unreported-ui-failure`          | A catch displays an error toast or sets error state without a recognized diagnostic in that branch.                                      |
| `local-only-ui-failure`          | A handled UI failure has a console or local Query report. Vector's browser log exporter is for local development.                        |
| `swallowed-detached-rejection`   | A promise explicitly detached with `void` has an empty rejection handler. Check that the operation reports failures before this handler. |
| `unclassified-server-failure`    | An error branch converts a checked Go error into a generic 5xx without a recognized operation-level diagnostic.                          |
| `weak-server-failure-diagnostic` | A Go error branch logs or traces before a generic 5xx but the event does not refer to the checked error.                                 |

The frontend rules use Oxlint's syntax tree for the web, admin, and public space apps
and shared browser packages. They inspect local `catch` blocks and inline Promise
rejection callbacks. They skip ordinary form submission and Query cache maintenance
for the detached-promise rule, plus the existing one-way project-access diagnostic POST
whose rejection would otherwise produce a second diagnostic about the exporter itself.
The Go rules use the standard Go parser for the API and its domain packages. They inspect
explicit error guards, 5xx statuses, and simple same-file response constructors. They
skip explicit cancellation branches and leave expected 4xx responses out of scope.

Treat results as review candidates. The audit does not follow calls across files, infer
coverage from a Query hook or middleware, or prove telemetry delivery. The Go HTTP
middleware already records generic server error responses, so Go findings call for a
closer look at the operation, stage, and cause classification. Check that any existing
span or shared logger covers the same failure before adding another signal. When adding
telemetry, use stable operation identifiers, a safe error class, and an approved request
ID where available. Keep credentials, user content, and raw response details out of
diagnostic payloads. See [Vector's telemetry configuration](self-hosting.md#logs-traces-and-metrics).

Run the rule fixtures with the pinned Bun and Go tools.

```bash
source scripts/dev-env.sh
bun test scripts/quality/observability/frontend.test.mjs
go test scripts/quality/observability/scan-go.go scripts/quality/observability/scan-go_test.go
```
