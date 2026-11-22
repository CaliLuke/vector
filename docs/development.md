# Development exceptions

Use [AGENTS.md](../AGENTS.md) for working agreements and the [README](../README.md#run-locally) for startup.
Keep this guide limited to non-obvious instructions that are absent from code, configuration, and those documents.

## JavaScript tooling runtime

All JavaScript applications, quality scripts, tests and supervisors run with Bun,
pinned in `.bun-version` and `.mise.toml`. CI and Docker use the same version.
Use `bun install --frozen-lockfile` and `bun run <script>`. `bunfig.toml` forces
Bun for dependency executables even when their shebang names Node.
Every JavaScript workspace links to that config because Bun loads it from the
command's working directory. The runner checks verify the runtime in each workspace.
Frontend behavior tests run in isolated Bun processes. The normal run exercises
built package exports; the coverage run resolves those exports to workspace source
so bundling cannot discard executable source mappings. CommonJS fixtures and ESM
imports share that resolution. Oxc collects Istanbul-format coverage, including
unloaded source files. Vitest suites use Oxc too; do not use Babel for coverage.
Frontend coverage baselines enforce both percentage floors and uncovered-count
ceilings; see the [migration review](bun-coverage-migration.md).
Node-compatible module imports and type packages do not require a Node runtime.

Knip's workspace-specific `ws` and `@types/ws` ignores track
[Knip #2294](https://github.com/webpro-nl/knip/issues/2294): Bun lists `ws` as a
built-in, so Knip misses its package and type references. Remove the ignores when
Knip recognizes those references under Bun.

## Git hook environment

`check.sh` clears Git's checkout-local environment after entering the repository root.
Linked-worktree hooks can export their Git directory and index. Test fixtures use
separate repositories and need Git to discover each fixture from its own directory.

## Cancel checks

Cancel through the launching terminal with Ctrl+C, or signal the invocation's process group.
Signaling only Prek does not reach detached workers.

## Full-check progress and reuse

`./check.sh --full` and the full Prek group run affected tests before expensive
checks and stop at the first failure. Go runs affected packages and their reverse
production/test imports first, then every remaining package exactly once in the
same integration coverage run. Shared fixtures or an unmappable deleted package
use the complete package order. There is no separate Go preflight that repeats
those tests. Frontend source changes run the existing behavior suite before builds.
Set `VECTOR_CHECK_BASE` to a Git revision when checking committed changes rather
than the working tree against `HEAD`.

Successful stages can be reused within the current Codex chat for eight hours.
Reuse requires matching source, configuration, dependency manifests, environment
and tool versions, plus intact output artifacts. A Go-only fix does not rerun
unchanged frontend builds or coverage. CI and `./check.sh --full --fresh` always
execute every required stage. Outside Codex, reuse is off unless the shell sets
`VECTOR_CHECK_SESSION` to an identifier for the current task.

The comparison base invalidates changed-file selection, not complete test/build
results. Updating the backend debt catalog still runs formatting but does not
invalidate frontend execution. Other documentation remains an input until its
consumers are reviewed.

Manual checks and Husky use the same repository-local tool path. Committing a
checked tree does not invalidate its results; source and configuration changes
still invalidate the affected stages.

Use `./check.sh --status` for the current stage, elapsed time, results and log path.
Logs and success records live under ignored `.quality/reports/stages/`.
A full result still requires every configured stage; focused tests alone do not
satisfy integration or coverage gates. Changing inputs during a run invalidates
its result.

## Docker Desktop on macOS

Use Docker Desktop for the application stack and integration tests. Install it
from Docker’s official distribution and finish its first-run setup. The local
startup script opens Docker Desktop if its engine is stopped and waits up to
two minutes for readiness.

Before the first run after switching from Colima, clear old shell overrides:

```sh
unset DOCKER_HOST DOCKER_CONTEXT TESTCONTAINERS_HOST_OVERRIDE LIVE_TEST_REDIS_BIND_HOST
source scripts/dev-env.sh
docker info
bun run dev
```

On macOS, `scripts/dev-env.sh` defaults to Docker Desktop’s user socket at
`~/.docker/run/docker.sock` and loopback addresses for published test ports.
Explicit `DOCKER_HOST` or `DOCKER_CONTEXT` selections remain supported; Linux CI
keeps its existing engine. Source this script in the shell that runs Git hooks too.
No dedicated Colima VM, SSH forwarding or reserved port range is required.

Docker engines have separate volumes. Move existing application data while the
source stack is stopped, preserve volume names and ownership, and verify the
restored database and object store before retiring the source volumes. Keep other
projects’ containers and Docker context unchanged. Do not start an empty stack
and treat it as a completed migration.

For a remote Docker engine, set `TESTCONTAINERS_HOST_OVERRIDE` to its reachable
address and `LIVE_TEST_REDIS_BIND_HOST` to a host address reachable from that engine.

## Recover a rejected type-level migration

For migration 46 failures, use the [journal recovery procedure](self-hosting.md#recover-a-rejected-type-level-migration).

## Reproduce transient UI states

Run a focused browser replay before the full quality gate. Build the workspace
exports once using `bun run build`, then run the replay.

```bash
bun run test:browser:status-filter
```

Set `PLANE_CHROME_PATH` if Chrome or Chromium is outside the usual install paths.
The replay bundles the current selector source with the built workspace packages.
It opens a fresh local Chromium profile against synthetic data. It holds a preferences response, delivers a
status rename first, then releases preferences. The JSON output lists the
response order and the selector's DOM state after React commits and mutations.
Rebuild `@plane/utils` after changes to its filter configuration with
`bun run --filter=@plane/utils build`.
It fails if an intermediate commit replaces known labels or disables the control,
even if the final DOM recovers. The replay needs no account or deployment.

Add one repeatable scenario per behavior. Start with cold and warm navigation,
then cover rapid filter edits, back/forward, mutations arriving out of order, and
a project switch with a request still in flight. Use explicit response gates so
the same schedule can be replayed against the current commit and a candidate.
The shared work-item query regression already checks a rapid pair of filter edits,
a superseded reply, replaced permissions, and a project switch. For a full-route
browser replay, seed two projects with distinct statuses and rows
and record loading, visible row IDs, and request IDs on each transition. Assert
that the latest filter wins, a refresh cannot manufacture an empty result, and a
late response from another project cannot update the current screen. Include an
explicit loading state when query scope or access changes.

Use request IDs to connect real API calls to SigNoz. For browser-only transitions,
the local timeline has the evidence even when every HTTP response succeeds.
Keep recordings limited to synthetic data. When a replay fails, add the smallest
test that can enforce the same invariant before batching changes for the full
quality run.
