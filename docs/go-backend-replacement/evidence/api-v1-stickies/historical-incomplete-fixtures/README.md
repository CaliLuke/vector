# Superseded API v1 Sticky captures

These six early captures omitted API audit tasks (and DELETE's deletion task).
They are retained as historical observations, not current parity proof. The
current corpus is `apps/go-api/internal/parity/testdata/fixtures/api_v1_stickies/`, recorded
from pinned source with real RabbitMQ consumption and complete effects.

The historical authentication-side-effect record is also retained here. Its old
audit representation lacks the canonical task args/kwargs envelope; the new source
corpus records the real consumed message. Audit-only full-backend metadata now
points here without regenerating that tree.
