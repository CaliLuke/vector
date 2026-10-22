# Remaining publication, preference, notification and provider corrections

The user requires source bugs to be fixed during migration. These changes use
pristine Django fixtures and separately captured corrected Django fixtures;
the replay compares Go with the corrected record only for the named cases.
Source revision and handler hashes are in each family contract.

- Publication PATCH/DELETE scopes the board to the route project and workspace.
  The source allowed a member to mutate another project's board by identifier.
  Foreign boards now return 404 without writes or queued deletion.
- Publication intake assignment accepts a valid intake identifier belonging to
  the route project. The source POST assigned the identifier to a model object
  property, crashed, and could leave a partially created board. Go validates the
  reference before saving; missing, deleted or foreign intakes return 400.
  PATCH also rejects foreign intakes. The corrected source validates the same
  reference and assigns `intake_id` before saving.
- Publication create/update rejects non-object JSON with 400. Preferences rejects
  non-object JSON with 400 and leaves the membership unchanged. These errors
  replace source exceptions or inconsistent serializer envelopes.
- Notification mark-all rejects non-object JSON with 400 before changing any
  notification. The source array request raised a server error.
- Notification mark-all OPTIONS returns ordinary action metadata. The source
  crashed while attempting to inspect a serializer that the action does not use.
- Unsplash queries use standard URL query encoding and the actual `page` value.
  The source interpolated `$page` and unescaped query text. Upstream success JSON
  and non-success JSON/status remain intact.
- AI requests require an object with string task/prompt fields after existing
  authorization and configuration checks. The source crashed on arrays, numeric
  tasks or a missing prompt. Corrected cases return 400 without upstream calls.

The optional `UNSPLASH_API_BASE_URL` deployment setting defaults to
`https://api.unsplash.com`. It accepts an HTTP(S) base without credentials,
query or fragment. `OPENAI_BASE_URL` uses the OpenAI SDK's existing setting.
Local replay runs a fixture HTTP server and the real SDK transport; no paid
provider calls are required.

The unchanged source behavior, including notification filter semantics, role
precedence, durable deletion admission and provider error envelopes, remains
covered by the canonical fixture corpus. Final proof, independent review and
live activation are tracked separately; this document does not assert release.

- User email update and verification-code generation reject a top-level JSON
  array with HTTP 400 and no database, cache, or task effects. The pristine
  handlers call `.get` on the list and raise HTTP 500. The corrected source
  checks the body type first and returns the endpoint's existing missing-email
  validation response: `{"error":"Invalid email format"}` for email update and
  `{"error":"Email is required"}` for code generation. Pristine and corrected
  records are kept separately in the user-account-write fixtures.

- App and space forgot-password reject a top-level JSON array with HTTP 400 and no task or cache effects. Pristine Django calls `.get` on the list and returns HTTP 500; corrected source returns the existing missing-email validation response. The app and space pristine fixtures remain unchanged, and the exact 500-to-400 body projections are limited to those two named differential cases.
