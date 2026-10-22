# HISTORY-001: Return merged issue history

Django chains activity and comment querysets and then indexes model instances as
dictionaries. This crashes with HTTP 500 for unfiltered mixed history, including
HEAD. Go serializes each queryset with the source projection and returns the
chronologically merged records with HTTP 200. HEAD returns the same status and
headers without a body. Authorization and read-only effects are preserved.

The separate pristine and corrected fixtures in
`apps/go-api/internal/parity/testdata/fixtures/work_item_history{,_corrected}` retain the
original failure and the corrected mixed-history and HEAD projections. The source
patch and capture receipts are in `evidence/work-item-history`.
This follows the standing instruction to correct source bugs.
