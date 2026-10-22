# HISTORY-002: Reject invalid history cursors

Both version-list endpoints return HTTP 400 with
`{"detail":"Invalid cursor parameter."}` for malformed cursors, non-positive
page sizes and offsets that cannot be represented. Django raises HTTP 500 for
malformed and zero-size cursors. The target also prevents division by zero and
overflow before querying a page. Existing authorization and missing-issue lookup
precedence remain intact; valid Python integer syntax uses the shared helper.

Pristine and narrowly corrected source fixtures record malformed cursors on both
lists and a zero-size issue-version cursor. Unit tests also cover invalid limits
and overflow. This follows the user's approved 400 policy for invalid pagination.
