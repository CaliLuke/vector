# Dependency selection and constraints

The manifests and lockfiles define the installed versions. Use
[pnpm-workspace.yaml](../pnpm-workspace.yaml), [pnpm-lock.yaml](../pnpm-lock.yaml),
and [go.mod](../apps/go-api/go.mod) when preparing another update. Git records
completed updates and verification; this document records current constraints.

## Selected families

| Family                                  | Selection                                              |
| --------------------------------------- | ------------------------------------------------------ |
| Node.js / pnpm                          | 26.12.1 / 12.15.0                                      |
| React / React DOM                       | 19.3.1                                                 |
| React Router                            | 8.5.1                                                  |
| Tiptap                                  | 3.32.2; retain the scoped collaboration override below |
| Hocuspocus                              | 4.8.0                                                  |
| Vite / Storybook                        | 8.3.5 / 10.6.5                                         |
| Vitest / coverage-v8                    | 5.0.5                                                  |
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
  compatibility API used by tools and test loaders. Follow the
  [side-by-side compiler guidance](development.md#requirements) before removing it.
- TanStack React DB stays on stable 0.5.12 with query-db-collection 1.4.8.
  Review peer ranges and migrate the bindings together before changing this
  cohort. A release candidate alone is not a reason to replace it.
- `react-popper@2.3.0` in Space and `use-font-face-observer@1.3.0` in Propel
  declare React peers through 18. Their React 19 peer warnings remain visible.
  Replace their integrations or adopt compatible declarations before closing
  this compatibility debt. No peer override hides the warnings.
- Testcontainers core and PostgreSQL share the upstream commit containing the
  [HostPort timeout fix](https://github.com/testcontainers/testcontainers-go/pull/4055)
  for [issue #4054](https://github.com/testcontainers/testcontainers-go/issues/4054).
  Replace both pins with a tagged release containing the fix. Preserve the
  existing proxy readiness deadlines; see [test commands](development.md#tests).
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
  Happy DOM, Form devtools, and transitive security pins. The
  [development limitations](development.md#dependency-limitations) retain the
  related upstream reports and consumer checks. ProseMirror packages must share
  type identities; broad transitive overrides must not force unrelated majors.

## Validation

Check available JavaScript updates with `pnpm outdated -r --format json` and
review prereleases and peer ranges before selection. For Go commands, first
source `scripts/dev-env.sh`. Validate updates in this consumer with affected
regressions and the complete repository quality gate described in
[development checks](development.md#quality-gates).

Frontend coverage is measured by the native workspace suites and the source-mapped
frontend behavior suite. [COV-03](tech-debt-catalog.md) records remaining gaps;
passing the coverage ratchets does not mean that all behavior is covered.
Keep dependency development within this repository's authorized scope.
