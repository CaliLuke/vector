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

## Colima test VM

Use a dedicated test VM with SSH forwarding.
Colima's direct VM address can lose TCP connection attempts under churn
([Colima #1664](https://github.com/abiosoft/colima/issues/1664)).
A transient native port collision can also leave an SSH-forwarded port unreachable
([Colima #1660](https://github.com/abiosoft/colima/issues/1660)).

Reserve a guest publish range outside macOS's ephemeral client-port range.
Check the macOS range with `sysctl net.inet.ip.portrange.first net.inet.ip.portrange.last`.
The configuration below assumes that range starts at 49152.

For one-time setup, create and stop the dedicated profile:

```sh
colima start vector-mcp-tests --cpu 2 --memory 3 --disk 15 --port-forwarder ssh --activate=false
colima stop --profile vector-mcp-tests
```

Merge this provision into `~/.colima/vector-mcp-tests/colima.yaml`. Keep any other provisions.
The provision sets the guest range before Docker starts:

```yaml
provision:
  - mode: after-boot
    script: sudo sysctl -w 'net.ipv4.ip_local_port_range=20000 29999'
```

For subsequent runs, start the configured VM and run checks:

```sh
colima start --profile vector-mcp-tests --activate=false
export DOCKER_HOST="unix://$HOME/.colima/vector-mcp-tests/docker.sock"
export TESTCONTAINERS_HOST_OVERRIDE=127.0.0.1
export LIVE_TEST_REDIS_BIND_HOST=127.0.0.1
source scripts/dev-env.sh
./check.sh --go --full
```

Use those exports in the shell that runs Git hooks too.
Keep native services out of ports 20000–29999 while this VM runs.
Leave the default VM and Docker context unchanged.
Before returning to the application's default stack, unset all three variables.
After testing, stop the VM with `colima stop --profile vector-mcp-tests`.

After an upstream fix, recheck the direct-address reproduction before removing this test-VM requirement.
Remove the range constraint only after forwarding recovers from transient port collisions.

For a Docker VM reached through its private address, set `LIVE_TEST_REDIS_BIND_HOST` to that address alongside `DOCKER_HOST`.

## Recover a rejected type-level migration

For migration 46 failures, use the [journal recovery procedure](self-hosting.md#recover-a-rejected-type-level-migration).
