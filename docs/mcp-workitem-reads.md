# Work-item comment, link, and activity reads

`workitem_comment`, `workitem_link`, and `workitem_activity` each expose `list`
and `retrieve`. Both actions require `project_id` and `workitem_id`; retrieve also
requires `comment_id`, `link_id`, or `activity_id`. Lists accept `cursor` and
`per_page`. Zero or omitted page size uses 1000. Workspace selection follows the
same explicit selector and authenticated context rules as other project reads.

These actions reuse the primary work-item collaboration and activity readers.
Their optional immutable workspace binding causes the data query to recheck the
active account, workspace, current workspace and project memberships, project,
and parent work item in one database snapshot. Restricted guests can read their
own items or items in projects that enable guest visibility. Deleted or draft
parents and archived projects are hidden. A visible parent with no children
returns an empty page; a hidden parent returns an error. Existing HTTP callers
retain their unbound behavior.

Comment and link pages preserve the pinned SDK's twelve page fields. Activity
pages use the toolkit's seven-field envelope. Comments expose the SDK's nineteen
fields plus the existing API's `description` field. Links have twelve fields;
activities have eighteen. Nullable fields remain explicit nulls. Activity epochs
must be exactly representable as signed 64-bit integers. Activity lists retain
the existing exclusion of comment, vote, reaction, and draft events.

Comment mentions become canonical `@[uuid]` tokens, preserving bytes outside the
mention spans. The checked-in golden fixture records outputs from the pinned
handler, including malformed markup, duplicate attributes, and raw-text elements.
Related comment, parent, and description IDs are checked against their owning
item, project, and workspace. Timestamps use the authenticated actor's timezone;
opaque link metadata is preserved.

Loom [#1037](https://github.com/CaliLuke/loom/issues/1037) tracks inconsistent
nullable-enum validation in untagged results. The generated nullable comment
`access` field temporarily omits its enum constraint; the application projection
still rejects non-null values other than `INTERNAL` and `EXTERNAL`. Restore the
generated enum when a released fix makes record and union validation agree.
No dependency source is patched.

Regression tests cover the generated MCP server under PAT and generic OAuth,
both supported protocol revisions, pagination, mention output, current
permissions, foreign relationships, and empty versus missing results. Local
browser/proxy verification and the rest of the accepted catalogue remain open.
