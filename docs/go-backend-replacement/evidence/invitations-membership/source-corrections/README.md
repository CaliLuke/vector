# ACCESS-001 corrected-source evidence

The corrected-source patch was generated from base commit `884590b2a2c325ffeb51ce3c58af6e315b841be4`, following [ACCESS-001](../../../decisions/ACCESS-001-invitation-membership-corrections.md). It contains the six corrected Django modules, including `plane/api/urls/invite.py` for the `include_format_suffixes=False` router decision.

The Go parity helper reconstructs pristine Django source with `git archive`, applies this patch to a separate temporary tree, then copies the regression tests and immutable pristine/corrected fixture corpora from the checkout. `source-inputs.SHA256SUMS` fingerprints every corrected source module and copied regression test; the helper verifies those hashes after reconstruction. It has no dependency on a machine-specific source path or on Django handler files remaining in the checkout.

`access001-decision-regressions.log` records eight passing corrected-source checks, including invalid role-5 PATCH preservation, PATCH rollback, DRF choice edges, and case-insensitive workspace-role caps. The injected `IntegrityError` produces the expected server log while the source view maps it to the tested 400 response.

The helper receipts are separate: `auth-callback-pristine.log` records the missing `ProjectMember.project_id` failure; `auth-callback-corrected.log` records the surviving helper regression after the bounded correction. Raw process exit codes are beside their logs.

The corrected JSON-suffix fixture records a 404, no changed application tables (including API token and activity-log rows), and exactly one durable `plane.bgtasks.logger_task.process_logs` delivery. Go replay compares this task against the source fixture. These checks do not replace full corrected HTTP parity, proxy differential, or final release proof.
