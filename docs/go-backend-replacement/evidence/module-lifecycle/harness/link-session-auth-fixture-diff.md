# Module-link session-auth corrected fixture diff

Capture probe, 8-case capture, and 8-case immutable verify passed. Fixture promotion happened only after all three stages passed.

Compared each promoted corrected JSON with its pre-capture snapshot outside the worktree. Every record has exactly two field changes: `/request/cookies/session-id` is now the recorder-redacted authenticated session cookie, and `response.headers.Vary` includes `Cookie`. Response status/body, database before/after/changed tables, and effects are unchanged for all eight cases.

| Fixture | Vary before | Vary after |
| `post-missing-module-foreign-key` | `origin` | `Cookie, origin` |
| `post-same-workspace-foreign-project-module` | `origin` | `Cookie, origin` |
| `post-foreign-workspace-module` | `origin` | `Cookie, origin` |
| `post-duplicate-url-in-foreign-module-scope` | `origin` | `Cookie, origin` |
| `post-deleted-at-input` | `Accept-Encoding, origin` | `Accept-Encoding, Cookie, origin` |
| `patch-deleted-at-input` | `Accept-Encoding, origin` | `Accept-Encoding, Cookie, origin` |
| `put-deleted-at-input` | `Accept-Encoding, origin` | `Accept-Encoding, Cookie, origin` |
| `post-create-on-deleted-module` | `origin` | `Cookie, origin` |
