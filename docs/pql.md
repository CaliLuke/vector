# Shared work-item query language

## Summary

PQL belongs to Vector's work-item query boundary. HTTP and MCP must execute the same predicates against the same authorized candidate set. MCP must not translate PQL into client-side filtering or publish a reference for language that cannot execute.

This design establishes the parser and scalar-query boundary. It does not settle the missing milestone or teamspace models and does not complete VEC-12 or VEC-22. The accepted target remains the full language in the [pinned reference](https://github.com/makeplane/plane-mcp-server/blob/dc7cfd9f3c305891909e2372f0cc507291353801/plane_mcp/pql_reference.py). The initial execution surface below is explicitly smaller than that target; it adds no MCP action or reference-tool registration.

## Problem statement and current state

Before this change, `internal/workitems/list.go` rejected all API `pql` and structured `filters` inputs. Application filters already use `internal/issuequery`; custom-property filtering additionally uses authorized property-discovery snapshots in `internal/workitemtypes`. Reimplementing those responsibilities inside MCP would produce different permission and query behavior.

## Proposed design

`internal/issuequery` owns PQL syntax and parameterized predicates. `internal/workitems` owns authorization, candidate selection, transaction scope, pagination, and projection. The compiler accepts an actor ID, one request-time instant, and a validated requester timezone. It emits only a predicate over the existing `issue` and `project` aliases plus bound arguments. It never selects rows, creates SQL identifiers from user input, performs authentication, or widens the candidate set.

The parser consumes the entire input before compilation succeeds. Each standalone predicate counts as one condition. Every expression is compiled, including logically unreachable branches. Unknown names, unavailable capabilities, invalid operands, and trailing tokens return HTTP 400 without executing the item query. Errors identify the byte offset and category; they do not echo literal values or query text. Compiler failure leaves the caller's argument slice unchanged.

| Capability         | Initial execution                                                                         | Rejection and remaining scope                                                                                                             | Default and equivalence                                                                                 |
| ------------------ | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Logic              | Parentheses, NOT, AND, OR; precedence in that order                                       | More than five conditions; malformed or trailing syntax                                                                                   | Blank PQL adds no predicate; count each comparison once, including IN and BETWEEN                       |
| Strings            | `priority`, `title`, `id`                                                                 | Unknown fields; invalid priority enum; ordering on strings                                                                                | Exact case-sensitive equality; `~` uses literal case-insensitive containment, except `id` uses a prefix |
| Booleans           | `isDraft`, `isArchived`                                                                   | Quoted booleans; non-boolean operands                                                                                                     | Bare true/false; predicate calls are equivalent to equality with true                                   |
| Dates              | `dueDate`, `startDate`, `createdAt`, `updatedAt`                                          | Invalid calendar values, date arithmetic, wrong function arity                                                                            | Requester timezone; one captured instant for all relative functions                                     |
| UUIDs              | `project`, `createdBy`                                                                    | Names instead of UUIDs; function on an incompatible field                                                                                 | UUID normalization; `createdBy = currentUser()` uses the authenticated actor                            |
| Operators          | `=`, `!=`, IN, NOT IN, null tests; text empty tests and `~`; date comparisons and BETWEEN | Operator/type mismatches; empty lists; NULL as a comparison operand                                                                       | BETWEEN is inclusive; both `a AND b` and `(a, b)` range forms parse                                     |
| Existing API lists | Apply PQL before count and pagination, including archived lists                           | Structured `filters` remain rejected; any nonempty external identity parameter combined with nonblank PQL is rejected                     | Absent PQL preserves existing list and external-identity behavior                                       |
| Remaining PQL      | No initial execution                                                                      | Relations, state groups, assignments, custom properties, remaining predicates, milestones and teamspaces require their domain integration | Reject explicitly; never drop an unsupported condition                                                  |
| MCP surface        | None added by this foundation                                                             | Core query actions and `get_pql_reference` remain unadvertised until their complete contracts execute                                     | Auth/protocol variants eventually share the same application entrypoint                                 |

An independent behavioral review and focused re-check found the initial contract ready after making function arities and DST resolution explicit.

The compiler's field and function definitions are the executable authority for this initial subset. Names are case-sensitive; logical/operator keywords are case-insensitive. There are no aliases. The pinned reference remains the target for subsequent additions, not an assertion of current support. Existing structured-filter semantics are unchanged.

## Detailed behavior

### Grammar and limits

Values are JSON double-quoted strings, bare booleans, decimal numbers, or function calls. Fields use their documented camelCase names; custom-field syntax can be parsed but must be rejected until an authorized property resolver is connected. Single-quoted strings, comments, and arbitrary expressions are rejected. No expression is evaluated as SQL or source code.

The input is limited to 16 KiB, nesting to 64 levels, and explicit lists to 256 values. These are resource bounds in addition to the five-condition language limit. Oversized inputs fail rather than being truncated. Lists contain scalar values, not nested lists. Function argument lists use the same bound.

Scalar equality, inequality, ordering, and negation preserve PostgreSQL's three-valued logic: a missing scalar is unknown and does not match even under NOT. Null tests are the way to select missing values. `IS EMPTY` means an empty string, and is different from `IS NULL`; `IS NOT EMPTY` excludes null. For priority, the string `none` is a real enum value, not SQL NULL. Booleans support equality/inequality and null tests, not IN or ordering. Dates support comparison, BETWEEN, and null tests, not IN or text operations. UUIDs support equality/inequality, IN/NOT IN, and null tests. Strings support equality/inequality, IN/NOT IN, null/empty tests, and containment; priority allows only its enum values and does not allow containment or empty tests.

`id` is the project identifier, a hyphen, and the item's decimal sequence ID. Equality compares that complete identifier; containment uses a prefix. `title` uses the item name. Date fields map to `target_date` and `start_date`; timestamp fields map to `created_at` and `updated_at`. Existing route rules for deleted, archived, draft, or epic items remain candidate restrictions: a PQL predicate never overrides them.

### Time

Empty timezone means UTC; an invalid nonempty timezone is an error. Date literals use `YYYY-MM-DD`; timestamp fields additionally accept RFC 3339 timestamps. Calendar years must be between 1 and 9999, including function results. A date literal on a timestamp field means midnight in the requester timezone. Comparisons do not silently expand equality into a day range.

The initial function allowlist is `now`, `today`, `startOfDay`, `endOfDay`, `startOfWeek`, `endOfWeek`, `startOfMonth`, `endOfMonth`, `startOfYear`, and `endOfYear` with zero arguments; `daysAgo`, `daysFromNow`, `weeksAgo`, `weeksFromNow`, `monthsAgo`, and `monthsFromNow` with one integer argument; `currentUser()` only as a value for `createdBy`; and standalone zero-argument `isDraft()` / `isArchived()`. All other function names and arities are rejected. Value functions are not standalone predicates.

`now()` is the captured instant. `today()` and `startOfDay()` are local midnight. Weeks begin Monday. Starts of week/month/year are local midnight; ends are one microsecond before the next period starts, matching PostgreSQL timestamp precision. Relative day/week/month functions apply calendar shifts to the captured local instant. Month shifts clamp the day to the destination month's final day. Integer offsets are nonnegative and at most 10,000; fractional offsets and other arguments are rejected. On date fields, date functions use the resulting local calendar date. DST therefore changes elapsed duration without changing calendar boundaries. For a computed civil time in a clock gap, reject the query rather than silently normalizing its date or clock time. For a fold, use the earlier instant. The same rule applies to literal local midnight and calendar boundaries. Relative shifts retain the local clock time and microseconds; all timestamp operands are truncated to microseconds. PQL timezone errors do not change the legacy serializer’s UTC fallback behavior.

### Routing and safety

```text
authorize the existing API request
reject structured filters and conflicting external-identity/PQL mode
parse the whole PQL expression with resource limits
compile every leaf with typed values and bound parameters
AND the compiled predicate with the authorized candidate restrictions
count and page those candidates through the existing list path
project through the existing HTTP serializer
```

The initial integration does not authorize MCP to call the current general list path: that path still needs immutable workspace binding and a consistent authorization/projection snapshot before MCP core reads are registered. Domain resolvers for relationships and custom properties must use that same snapshot and mask foreign or invisible references under both positive and negative predicates. The existing property-filter true/false/unknown model is the relevant boundary; this design does not replace it.

## Proof obligations

- Precedence, grouping, both range forms, and five-condition accounting select the intended records; invalid branches cannot be hidden behind OR or NOT.
- Literal quotes, SQL-looking text, and LIKE wildcard characters remain data. Failed compilation cannot partially modify query arguments.
- Date boundaries hold across DST, leap years, month-end shifts, and non-UTC requests.
- Public repository lists filter before count and page limits, preserve legacy behavior without PQL, and never return another project's items because a predicate names that project.
- Every currently executable capability has a public compiler or real PostgreSQL repository proof. The parity tracker continues to show all unimplemented actions; syntax support alone never changes its counts.

## Remaining domain decisions

Milestone and teamspace ownership, membership, archive behavior, and relationships require inspected domain designs before their resolvers exist. Property PQL requires an explicit mapping from the pinned property type/operator matrix into Vector's effective property catalog. Those are remaining work, not intentionally excluded full-parity features. The full reference tool must wait for all documented examples and resolver dependencies to work.
