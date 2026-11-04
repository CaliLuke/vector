# Dependency selection and constraints

The manifests and lockfiles define the installed versions. Use
[package.json](../package.json), [bun.lock](../bun.lock),
and [go.mod](../apps/go-api/go.mod) when preparing another update. Git records
completed updates and verification; this document records current constraints.

## Selected families

| Family                                  | Selection                                              |
| --------------------------------------- | ------------------------------------------------------ |
| Bun                                     | 1.4.4                                                  |
| React / React DOM                       | 19.3.1                                                 |
| React Router                            | 8.5.1                                                  |
| Tiptap                                  | 3.32.2; retain the scoped collaboration override below |
| Hocuspocus                              | 4.8.0                                                  |
| Vite / Storybook                        | 8.3.5 / 10.6.5                                         |
| Vitest / Oxc coverage                   | 5.0.5 / 0.19.0                                         |
| TypeScript compiler / compatibility API | 7.0.2 / 6.0.2                                          |
| Oxlint / Oxfmt / Knip                   | 1.90.0 / 0.75.0 / 6.44.0                               |
| Go                                      | 1.27.3                                                 |
| Loom / Loom-MCP                         | 1.10.0-alpha.22 / 2.1.0-alpha.50                       |
| River library / pgx driver / CLI        | 0.51.0                                                 |
| Testcontainers core / PostgreSQL        | 0.44.1-0.20261029064637-ff84c26717f1                   |

Keep related package families aligned. Update River's CLI and foundation pin
with its library. Regenerate only affected source designs when adopting Go
transport changes; do not edit generated transports manually.

## Retained constraints

- Typechecks use TypeScript 7. The separate TypeScript 6 alias supplies the
  compatibility API used by tools and test loaders. Keep the compiler and
  compatibility API separate during updates.
- TanStack React DB stays on stable 0.5.12 with query-db-collection 1.4.8.
  Query DB Collection 1.5.0 requires `@tanstack/db@^1.0.0-rc.0`; its
  [release notes](https://github.com/TanStack/db/blob/main/packages/query-db-collection/CHANGELOG.md#150)
  explain the stable version number on the release-candidate cohort. Review peer
  ranges and migrate the bindings together before changing this cohort. A release
  candidate alone is not a reason to replace it.
- `react-popper@2.3.0` in Space and `use-font-face-observer@1.3.0` in Propel
  declare React peers through 18. Their React 19 peer warnings remain visible.
  Replace their integrations or adopt compatible declarations before closing
  this compatibility debt. No peer override hides the warnings.
- CEL remains on `github.com/google/cel-go` 0.31.0. Version 0.32.0 changes
  its module path to `cel.dev/cel-go`; see the [upstream migration notice](https://github.com/cel-expr/cel-go/releases/tag/v0.32.0).
  Adopt that path when the importing dependencies migrate. Do not replace the
  old module path with a module that declares a different identity.
- Testcontainers core and PostgreSQL share the upstream commit containing the
  [HostPort timeout fix](https://github.com/testcontainers/testcontainers-go/pull/4055)
  for [issue #4054](https://github.com/testcontainers/testcontainers-go/issues/4054).
  Replace both pins with a tagged release containing the fix. Preserve the
  existing proxy readiness deadlines.
- `tiptap-markdown@0.9.0` needs a package extension declaring its runtime
  `@tiptap/pm` peer. [Upstream issue #88](https://github.com/aguingand/tiptap-markdown/issues/88)
  tracks the missing declaration. Remove the extension when the package declares
  that dependency and the editor integration passes without it.
- Tailwind uses `tailwindcss/index.css` because Vite fails to resolve the bare
  CSS import in SSR builds. [Vite issue #23096](https://github.com/vitejs/vite/issues/23096#issuecomment-6356112374)
  contains the reproduction. Restore the package-root import when both client
  and SSR builds pass without the workaround.
- Express 5 routers use `express-ws.applyTo` explicitly. The default prototype
  patch misses those routers; see [express-ws issue #169](https://github.com/HenningM/express-ws/issues/169).
  Revisit this initialization when upstream supports Express 5 routing by default.
- The workspace override comments retain removal conditions for collaboration,
  Happy DOM, Form devtools, and transitive security pins. Before removing the
  collaboration override, also run the page typing/reload integration check.
  ProseMirror packages must share type identities; broad transitive overrides must not force unrelated majors.

## Dependency limitations

- **PostgreSQL cleanup:** A client paused during SCRAM authentication can block `DROP DATABASE`, even for another database.
  The [upstream report](https://www.postgresql.org/message-id/179223137070.3783589.15732839710523354747%40gmail.com)
  records this barrier wait. Do not infer the cause of a preceding transport failure from a cleanup delay.
  Recheck this limitation after an upstream release processes barriers during authentication waits.
- **Router CommonJS warning:** Do not suppress the `replaceRouteChunk` circular-access warning or patch the dependency.
  After adopting a fix for [TanStack/router#8585](https://github.com/TanStack/router/issues/8585#issuecomment-6540692152),
  recheck both CommonJS and ESM imports plus CLI route generation.
- **Multipart errors:** Keep `NextPart` errors separate from body-read errors.
  [errorlint #116](https://codeberg.org/polyfloyd/go-errorlint/issues/116) reproduces
  the error-reassignment false positive on 1.9.0; the related #49 is closed.
  Remove the variable-separation workaround when the pinned analyzer accepts that reproduction.
  Preserve direct `io.EOF` comparisons so wrapped EOF errors still report parse failures.
- **Go coverage overlays:** Do not combine source overlays with `-coverpkg`.
  First require a toolchain fix for [golang/go#73802](https://github.com/golang/go/issues/73802) and a passing reproduction.
  The defect can ignore overlaid code and produce a false passing test. Use applied source and a fresh build.
- **HTTP body cleanup:** Retain explicit checked closes until the pinned analyzer accepts the
  [cross-package helper reproduction](https://github.com/timakin/bodyclose/issues/60#issuecomment-6480673025).
  Remove decoder-factory lint allowances only after it also accepts the
  [factory reproduction](https://github.com/timakin/bodyclose/issues/71#issuecomment-6496543573).
- **Intentional error conversion:** Preserve validation and denial results at the retained nilerr allowances.
  Returning operational errors would change those API contracts. Remove allowances after the analyzer accepts the
  [nilerr #12 reproduction](https://github.com/gostaticanalysis/nilerr/issues/12#issuecomment-6543387375).
- **Tooltip refs:** Use Floating UI's returned ref setters directly only after the pinned analyzer accepts the callback-member form in
  [oxc#23753](https://github.com/oxc-project/oxc/issues/23753#issuecomment-6510863595).

Lagotto is an optional one-off structural audit tool. It is not a product dependency,
required quality gate, or ongoing debt item.

## Backend upstream check — 2026-11-04

- Testcontainers: latest tag is still 0.44.0. HostPort PR #4055 is merged and
  already included in our commit pin; no tagged replacement exists. Ryuk reuse
  PR #3868 remains open, so the reconnection grace remains necessary.
- PostgreSQL: 18.6 is still the latest stable 18.x release and is already in use.
  The linked barrier discussion does not establish a released authentication-wait
  cleanup fix; keep the cleanup observation open.
- Go: 1.27.3 is current and already pinned. Issue #73802 remains open with a
  reproduction on that version; there is no release to adopt for overlay coverage.
- Analyzers: the repository already uses golangci-lint 2.15.0 with errorlint
  1.9.0, nilerr 0.1.2 and bodyclose commit 857993a2939c. The bodyclose and nilerr
  reports remain open with reproductions on these versions. The multipart
  reassignment case still reproduces on errorlint 1.9.0 and is now tracked in #116.

## Validation

Check available JavaScript updates with `bun outdated` and
review prereleases and peer ranges before selection. For Go commands, first
source `scripts/dev-env.sh`. Validate updates in this consumer with affected
regressions and the complete repository quality gate described in
[AGENTS.md](../AGENTS.md).

Frontend coverage is measured by the native workspace suites and the source-mapped
frontend behavior suite. [COV-03](tech-debt-catalog.md) records remaining gaps;
passing the coverage ratchets does not mean that all behavior is covered.
Keep dependency development within this repository's authorized scope.
