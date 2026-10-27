# Work item types: implementation and UI review

Updated October 26, 2026. This replaces the October 24 prototype status.
The contract is [work-item-types-spec.md](work-item-types-spec.md); current gate
results and chronological review evidence live in
[work-item-types-completion.md](work-item-types-completion.md).

## Production entry points

- [Workspace library](http://localhost:8000/notion-sync/settings/work-item-types/):
  workspace types, property definitions, attachments, contexts, and cascading editors.
- [Project availability](http://localhost:8000/notion-sync/settings/projects/15a482d9-bde1-4820-b67e-0b2f3bc81b12/work-item-types/):
  shared workspace imports, availability, and project default.
- Native creation, update, draft, full detail and peek surfaces consume the same
  authorized definitions and values. Lists, boards, tables, filters and exports
  use authorized projections.

These production surfaces persist through the API and PostgreSQL. Retained
prototype scenarios support comparison only; they are not persistence evidence.
No preview warning banner or screen/state toolbar appears in the production UI.

## Acceptance coverage

Each row pairs production UI coverage with backend assertions. Passing a fixture
interaction alone does not establish persistence or authorization.

| Spec | Implemented behavior                                                    | Verification                                                                                                        |
| ---- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| S1   | Activation, protected Task, rollout, counts and active status           | Repository lifecycle, duplicate/stale/permission tests; live library reload                                         |
| S2   | Compact type editor and anchored icon/color sheet                       | Saved appearance and failed-save tests; live keyboard and 320px sheet checks                                        |
| S3   | Shared attachments, per-type ordering, scoped preservation              | Collision and detach-review integration tests; drag/keyboard UI tests and live persisted reorder                    |
| S4   | Eleven property kinds and formula authoring                             | Property validation and formula evaluation tests, including zero/false and error/empty distinctions                 |
| S5   | Context scope, inheritance, precedence and overrides                    | Persisted context/authorization tests; staged-edit UI assertions and narrow-screen interaction                      |
| S6   | Cascading levels, branch options and preservation review                | Path/ancestry validation, in-use deletion confirmation and draft retention tests                                    |
| S7   | Imports, pause/resume/removal, shared definitions and defaults          | Project integration tests, existing-item/draft safety and mandatory future-project coverage                         |
| S8   | Durable hierarchy editing, preview and reviewed conflict resolution     | Hierarchy integration and UI tests covering stale graphs, hidden identities, cancellation and idempotent commit     |
| S9   | Typed native forms, private drafts and atomic publication               | Form/Query tests paired with database publication, rollback, permissions and durable-effect assertions              |
| S10  | Type switching, detail/peeks, preserved values and rich-text history    | Reviewed-edit, restore and search integration tests; native detail/editor regression coverage                       |
| S11  | Typed filters, display columns, type/custom/parent grouping and exports | Predicate/grouping/projection tests, clock refresh tests, live grouping/table checks and a completed 400-row export |

## Visual and interaction evidence

The specification records the original documentation screenshot compositions
R1–R8 and distinguishes observed UI from proposed Vector designs. This final pass
verified local production behavior; it does not claim a fresh pixel comparison of
every screen/state against upstream media.

Recent local captures are ignored verification artifacts:

- `tmp/type-attachments-narrow.png`: attachment labels and controls at 390px.
- `tmp/type-icon-picker-narrow.png`: icon grid and color tiles on a narrow viewport.
- `tmp/property-context-narrow.png`: scoped context editor and fixed actions.
- `tmp/type-grouping-list.png`, `tmp/type-subgroup-board.png`: type grouping.
- `tmp/custom-property-grouping.png`, `tmp/parent-type-grouping.png`: additional axes.
- `tmp/custom-export-completed.png`: completed native export.

Keyboard checks covered Enter/Space activation, selected icon/color state, Escape
and focus return. A shared menu wrapper prevented keyboard actions from reaching
the actual button; forwarding its ref and role to the button fixed the defect.
The regression failed before the change, then all 30 dropdown tests passed, and
the live context action opened with Enter.

Temporary verification definitions and items were cleaned up, and original list
preferences were restored. The user's workspace hierarchy was not irreversibly
enabled merely to exercise it; database and UI tests cover the enabled builder.

## Release verification

Fresh independent implementation and acceptance reviews completed. Go full gates,
frontend production builds, typechecks, lint, formatting, duplication and coverage
passed. The combined run found an unused export and stale test assertions; after
correction, Knip and all 1,493 frontend tests pass. See the completion tracker for
exact run provenance and logs.

An issued download URL or downloaded export remains a snapshot; custom export
history rechecks current visibility before offering a new link.
