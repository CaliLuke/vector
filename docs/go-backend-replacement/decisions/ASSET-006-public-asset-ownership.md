# ASSET-006: preserve public uploads with owner-scoped mutations

Status: accepted security correction; focused owner-flow and PostgreSQL regressions pass.
Final composed proof and fresh review passed on repair commit `52f40b1738ab1c69fdb47826a246afe3c1f81591`; see the [final acceptance](../evidence/post-retirement-review/final-acceptance/acceptance.md).

Sites supports signed-in visitors who are not project members. Its file service
creates an upload, confirms it, and associates it with the visitor's comment.
Public uploads and reads remain supported. The unsafe source behavior permits
another signed-in visitor to confirm, delete, restore or reassign that upload.

Public mutation authorization must use the asset's recorded creator and the
resolved public project's workspace/project scope. Confirmation, delete and
restore must reject foreign assets before writes or metadata jobs. Comment
association must validate the selected assets and the target comment's actor and
project; mixed authorized/unauthorized selections must not partially mutate data.
A new asset's supplied comment reference needs the same target validation, while
an empty reference still permits upload before comment creation. Public board
resolution must enforce live, enabled and consistent project/workspace identity.

The change must preserve valid nonmember owner operations and published reads.
It must not introduce a blanket project-membership requirement. Historical
source fixtures remain intact. See [audit](../evidence/post-retirement-review/auth-effects.md)
and [campaign ledger](../post-retirement-review-ledger.md) for verification.
