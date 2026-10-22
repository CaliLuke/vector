# Project intake: return 404 for missing delete targets

Django dereferences a missing intake row and returns HTTP 500. The Go endpoint
returns HTTP 404 with `{"detail":"No Intake matches the given query."}` and
makes no database change. This fixes the source crash under the migration's
established bug-fix principle.

The [source decision and corrected captures](../evidence/project-intake-writes/decision-notes.md)
retain the pristine behavior and the corrected oracle. The contract projects
only the four named canonical DELETE cases for the inbox/intake aliases from
the exact recorded 500 status/body to the exact 404 status/body. Headers and
all other cases remain compared directly.
