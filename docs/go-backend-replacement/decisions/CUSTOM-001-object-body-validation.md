# CUSTOM-001: Reject non-object view preference bodies

The project and workspace view preference POST endpoints require a JSON object.
The pinned source reaches `.get()` on a top-level array and returns HTTP 500.
The migration returns HTTP 400 with `{"error":"The payload is not valid"}` after
authentication, timezone initialization, and the existing project/member lookup.
It performs no database writes or durable effects for these invalid bodies.

This follows the standing instruction to fix source bugs instead of preserving
crashes. Object payloads retain their existing behavior, including scalar values
inside `view_props`. Pristine fixtures retain the source 500; the two corrected
source fixtures add only object validation after the same authorization/lookups.
The corrected recorder invokes the original source method for valid objects.
