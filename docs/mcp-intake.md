# Intake API

Intake actions address the underlying work-item ID, not the intake association ID.
Native intake creation, triage transitions, deletion and durable activity remain
the application operations behind MCP. Lists retain cursor pagination and the
native snooze filter. Reads bind item visibility, current workspace/project
membership, the selected intake and related data to one database snapshot.

Current project members can submit items. Guests see their own submissions.
Project administrators can change triage and source metadata; other roles receive
an authorization error instead of a successful response with ignored fields.
Deletion retains the native rule: removing an accepted association leaves the
work item, while deleting an unaccepted submission requires its creator or a
project administrator. Every operation requires a current, unarchived project.
Duplicate targets must be live, visible work items in the same workspace.

Optional receipts recheck current membership and action rights before replay.
Revision tokens use the underlying item identity and advance for both item and
intake-association changes, including native writers. Creation and triage reuse
the existing transactional effects path. Deletion keeps native soft deletion and
cleanup behavior. These additions await the consolidated integration pass.
