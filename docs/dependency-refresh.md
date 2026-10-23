# Dependency refresh

This refresh aligns the primary runtime and toolchain on supported releases. It does not claim that every workspace dependency is at its newest release; retained versions and upgrade constraints are listed below.

## Major runtime and editor updates

| Area                   | Previous                      | Selected             | Notes                                                                                                                                                         |
| ---------------------- | ----------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| React / React DOM      | 18.3.1                        | 19.3.1               | Updated the shared catalog and React type packages to 19.3.0. React 19 requires source and declaration fixes for nullable refs and stricter component props.  |
| React Router           | 7.18.4                        | 8.5.0                | All `@react-router/*` packages and `react-router` are aligned. Current source has no `react-router-dom` imports or old `useMatches().data` reads.             |
| Vite                   | 8.0.16                        | 8.3.5                | App configs use native `resolve.tsconfigPaths`, so `vite-tsconfig-paths` was removed.                                                                         |
| Tiptap                 | 2.26.x                        | 3.32.x               | The `@tiptap/*` family is aligned; editor and Live consumers use v3 APIs and contract tests.                                                                  |
| Hocuspocus             | 2.x                           | 4.7.1                | Server, provider, transformer, and extensions are aligned.                                                                                                    |
| Blueprint core         | 4.16.3                        | 6.21.0               | The tooltip now comes from core; the separate React-18-only `@blueprintjs/popover2` package was removed.                                                      |
| Headless UI            | 1.7.19                        | 2.2.10               | Updated to a React 19-compatible release.                                                                                                                     |
| React Markdown         | 8.0.7                         | 10.1.0               | Updated with the React 19 runtime.                                                                                                                            |
| Storybook              | 10.4.6                        | 10.6.3               | Core, React renderers, and addons are aligned. The UI package now uses the Vite renderer and no longer carries webpack renderer addons.                       |
| Storybook Designs      | 11.1.3                        | 12.0.0               | The addon supports Storybook 10.6 and React 19.                                                                                                               |
| Tailwind CSS           | 4.1.x                         | 4.3.3                | Tailwind and its PostCSS integration use the same 4.3.3 release.                                                                                              |
| React Day Picker       | 9.5.0                         | 10.0.2               | Retains the supported react-day-picker package name.                                                                                                          |
| Node.js types          | 22.12.0                       | 22.20.5              | Kept the types aligned with the repository's Node 22 runtime.                                                                                                 |
| Atlaskit drag and drop | 1.7.4                         | 4.0.0                | Auto-scroll 3.2.1 and hitbox 3.0.0; consumers migrate to public subpaths.                                                                                     |
| Express                | 4.22.0                        | 5.3.0                | Live routes and middleware were migrated and validated on Express 5; its request-listen callback now handles errors as required by v5.                        |
| Helmet                 | 7.2.0                         | 8.3.0                | Live uses the default middleware configuration. The v8 default HSTS max-age is one year; the Live service is only exposed behind the TLS proxy in production. |
| dotenv                 | 16.4.7                        | 18.0.8               | Vite and Live only use `config()`. No consumer uses removed preload or `.env.vault` support, and all deployment images run Node 22.                           |
| React PDF              | 4.3.x                         | 4.9.0                | Renderer and type packages are updated together.                                                                                                              |
| Masonry                | react-masonry-component 6.3.0 | masonry-layout 4.2.2 | Replaced the unmaintained React wrapper with the direct layout library and its 4.2.8 type package.                                                            |

Other same-major updates include Axios 1.20.0, date-fns 4.4.0, UUID 14.0.3, i18next-icu 2.6.1, React Hook Form 7.90.0, SWR 2.5.1, tailwind-merge 3.7.0, Yjs 13.6.33, and y-protocols 1.0.7.

## TypeScript and quality tools

The native TypeScript 7 CLI is installed as `@typescript/native` (`npm:typescript@^7.0.2`). The ordinary `typescript` alias remains on 6.0.2 because the frontend Node test harness uses `transpileModule`, an API that must remain available while the native 7 API is being adopted. Oxlint uses its TypeScript 7 type-check integration; the full gate refuses to baseline compiler diagnostics.

Oxlint and Oxfmt are the lint and formatting tools. Knip findings and Oxlint diagnostics are stored as normalized, per-finding baselines so a new item cannot replace a different old item while keeping the same total. Live, codemods, editor, and UI tests report per-file V8 coverage; the Node frontend suite is listed as uncovered because its custom loader does not preserve usable source maps.

## Compatibility constraints

The dependency inventory is checked with `pnpm outdated -r --format json` and
recorded under `.quality/reports/`. Selected dependency majors now include Effect
4, Hocuspocus 4, Zod 4, Atlaskit core 4 and its auto-scroll/hitbox 3 packages,
TanStack Table 9, Vitest 5, jsdom 30, Lucide 1, Framer Motion 14, React Dropzone 20,
React Day Picker 10, i18next 26/react-i18next 17, and Storybook Designs 12.

- Node types stay on 22.20.5 to match the oldest supported deployment runtime.
  The supported runtime ranges are Node 22.22.2+, 24.15+, or 26+, matching jsdom's
  requirements. CI and deployment images continue to use Node 22.
- MobX 6.16.1 and mobx-react 9.2.2 stay aligned with mobx-utils 6.1.1, whose peer
  range requires MobX 6. MobX 7 and mobx-react 10 require an aligned mobx-utils
  release or replacement of its existing store integrations. The final audit
  found 60 source files using `computedFn`; its argument-based memoization and
  observer cleanup must be preserved, not replaced by an identity wrapper just
  to remove the peer constraint. See the [MobX computed-function guidance](https://mobx.js.org/computeds-with-args.html)
  and [mobx-utils package](https://github.com/mobxjs/mobx-utils).
- `react-popper` 2.3.0 and `use-font-face-observer` 1.3.0 have no newer releases
  and declare React peers only through 18. Their peer warnings remain visible.
  Replacing their APIs is separate migration work; no peer override hides them.
- The peer checker reports a Tailwind Typography mismatch despite its declared
  range accepting the installed Tailwind 4.3.3. No peer override is applied.

Unused direct dependencies were removed: Emoji Picker React from Web, the Effect
platform adapters from Live, TanStack Table from Propel, Zod from shared-state,
and postcss-nested from UI. The unused lint-staged dependency/configuration was
removed because Husky invokes Prek. The doctor command now uses the installed,
pinned react-doctor version instead of downloading an unpinned latest CLI.

## Overrides and package workaround

Workspace overrides are now version-scoped where a dependency has more than one supported major. In particular, `nanoid`, `chokidar`, `picomatch`, `glob`, `brace-expansion`, `postcss`, `express`, `axios`, and `rollup` no longer force a different major onto every transitive consumer. The Express 5 migration also scopes its v5 type adapter to `@types/compression` and `@types/express-ws`, whose wildcard type dependencies otherwise resolve to the old Express 4 declarations. The ProseMirror packages are pinned to the versions selected by Tiptap 3.32 so plugins share the same model, state, and view type identities.

`pnpm-workspace.yaml` extends `tiptap-markdown@0.9.0` with its missing runtime peer `@tiptap/pm` at `^3.0.1`. The extension is linked to [upstream issue #88](https://github.com/aguingand/tiptap-markdown/issues/88) and can be removed when the package declares this runtime requirement itself. Its presence was verified in the installed package's virtual-store `node_modules`.

The shared Tailwind stylesheet uses the explicit `tailwindcss/index.css` export
because Vite 8.3.5 fails to resolve the bare package CSS import in SSR builds.
A standalone client/SSR reproduction confirms the defect and the explicit-path
resolution; evidence is recorded in [Vite issue #23096](https://github.com/vitejs/vite/issues/23096#issuecomment-6356112374).
Remove the explicit-path workaround when the upstream fix ships and both client
and SSR builds pass with the package-root import.

## Go dependency refresh

Loom is updated from `v1.10.0-alpha.15` to `v1.10.0-alpha.16`. The 37 generation
manifests record the selected version; generated Go transports are unchanged.
The consumer's build, transport tests, and integration suite validate adoption.

The Go registry inventory identified 16 newer directly required modules. They are
now selected in `apps/go-api/go.mod`, with the corresponding transitive modules
and sums updated by Go. The separately versioned River CLI is also aligned with
the library, and the foundation pin check now checks both.

| Module family                    | Selected versions                                                           |
| -------------------------------- | --------------------------------------------------------------------------- |
| AWS SDK                          | core 1.47.4, config 1.33.10, credentials 1.20.10, S3 1.114.5, Smithy 1.28.6 |
| golang-migrate                   | 4.20.1                                                                      |
| OpenAI Go SDK                    | 3.86.0                                                                      |
| libopenapi / validator           | 0.41.6 / 0.16.1                                                             |
| PostHog                          | 1.36.0                                                                      |
| RabbitMQ AMQP client             | 1.16.0                                                                      |
| River library, pgx driver, CLI   | 0.49.0                                                                      |
| Testcontainers core / PostgreSQL | 0.44.0                                                                      |
| urfave/cli                       | 3.14.0                                                                      |

[River's release notes](https://github.com/riverqueue/river/blob/v0.49.0/CHANGELOG.md)
change transaction helpers and uniqueness behavior, but this consumer currently
uses River's client type and schema migrations, without enqueue or transaction
API calls. The intervening new migration applies to SQLite; this application uses
PostgreSQL. A new disposable-PostgreSQL regression checks fresh and repeat runtime
migrations, both migration ledgers, and preservation of a stored support-table row.

The [Testcontainers 0.43 changes](https://github.com/testcontainers/testcontainers-go/releases/tag/v0.43.0)
to SQL wait callbacks and custom image providers do not affect this consumer's
APIs. Its PostgreSQL, proxy, and broker tests exercise the updated container runtime.
[golang-migrate 4.20.1](https://github.com/golang-migrate/migrate/releases/tag/v4.20.1)
includes the corrected release packaging for 4.20.0; the existing pgx5 and embedded
migration source integrations remain in place.

## Further runtime and test-tool migrations

Effect 4 replaces the PDF service definition with an explicit Context service
and Layer, and uses the current schema, error recovery, timeout, and retry APIs.
Focused tests check request validation, timeout errors, exact retry counts, and
recovery behavior. Unused platform adapter dependencies were removed.
See the [Effect migration guide](https://github.com/Effect-TS/effect/blob/main/MIGRATION.md).

Vitest 5 and its V8 coverage package are aligned, with Vite declared directly by
all test workspaces. jsdom 30 drives the DOM tests. Go quality tools are pinned to
golangci-lint 2.15.0 and x/tools 0.52.0; the newer errcheck also checks discarded
errors received from channels. The crash-proof subprocess cleanup explicitly
ignores its expected killed-process exit error.

Visual review has exercised login, workspace navigation, persisted Tiptap content,
date selection, analytics rendering/filtering, and the public landing page. It
found sidebar transition, priority-option accessibility, analytics input control,
and collaboration transport defects. Focused regressions cover the repairs.
Final browser verification confirmed two-tab collaboration and synchronized
cleanup, persisted Tiptap formatting and due date, priority selection surviving
reload, and analytics filtering and clearing without React warnings.

After the approved Colima restart repaired new Docker port forwarding, the Go
full gate passed with 8,626 tests and 36 documented skips. The TypeScript full
group, production builds, and both Storybooks passed after adding coverage for
all three WebSocket binary representations and correcting the router's exposed
type. The final frontend behavior run passed all 165 tests. Evidence is in
`.quality/reports/prek-full-post-restart.log` (Go) and
`.quality/reports/prek-typescript-final.log` (final TypeScript/build rerun).

The UI review also observed repeated intake-state 404 logs. The local project
has no triage state, and the existing API intentionally returns 404 in that case;
the unchanged frontend retries this lookup. This remains a separate request/noise
cleanup, not a failed editor or analytics check. The development stack was stopped
after review; the other projects' previously running containers were restored.

After two memory-pressure crashes, the development stack was stopped. A guarded
reproduction isolated a DOM assertion-formatting runaway in a new transition
test. The test now waits for transition completion and uses boolean DOM identity
assertions. Tests and checks now run sequentially under external process-group
deadlines, sampled RSS ceilings, and Node heap limits; the development launcher
is guarded too. See the development guide for the limits.

The direct ioredis client is now v6. Hocuspocus's Redis extension retains its own
supported v5 client and receives connection options through its public API,
without a forced transitive override. A disposable-Valkey integration exercises
the application manager and cross-instance admin messages. Admin JSON has a
separate subscriber from the binary collaboration protocol.

Lucide's removed brand exports use react-icons while retaining saved icon names.
The UI and Propel catalogs share their 155 names/order; the UI retains its 12
custom glyph overrides. TanStack consumers use the native v9 API, and Atlaskit
consumers use supported public entry points.

Live imports editor helpers through the server-safe `@plane/editor/lib` entry
point. The editor's main entry point exports browser components and loads UI
dependencies that require a bundler; importing it from native Node caused the
collaboration server to exit during startup. Type-only imports remain erased.

The Express 5 router is explicitly initialized with `express-ws.applyTo`, using
the documented API. The default prototype patch fails for Express 5 router
instances; reproduction evidence was added to [express-ws issue #169](https://github.com/HenningM/express-ws/issues/169).
Revisit explicit initialization when default routing supports Express 5.
Hocuspocus 4 custom transports now forward message, close, and pong events.
Real WebSocket tests exercise authentication rejection, two-way edits, and
connection cleanup through the application router.
