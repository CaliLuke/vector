# Vector architecture

## Runtime

The Go service owns the application API, authentication, database bootstrap, and durable background jobs.
Caddy routes API requests to Go and serves the browser applications through the same origin.
PostgreSQL stores application data and durable work. Redis provides cache services.
S3-compatible storage holds uploaded files. The Node.js collaboration service supports live editing.

The runtime does not require Django, Celery, or RabbitMQ.
Static response fixtures remain for contract regression tests. No Python code or interpreter is required.

## Source layout

| Path                                   | Purpose                                       |
| -------------------------------------- | --------------------------------------------- |
| `apps/go-api`                          | Go API, workers, bootstrap, schema, and tests |
| `apps/web`                             | Main React application                        |
| `apps/admin`                           | Instance administration                       |
| `apps/space`                           | Public project pages                          |
| `apps/live`                            | Collaboration service                         |
| `apps/proxy`                           | Caddy configuration                           |
| `packages`                             | Shared frontend code                          |
| `apps/go-api/internal/parity/testdata` | Historical contract fixtures                  |

The browser applications retain substantial Plane-derived code. A future frontend rewrite is planned.
Existing API paths, database names, internal `@plane/*` imports, and configuration keys remain compatible.
The backend Go module retains its existing import path to preserve generated code.
These identifiers do not identify the product as an official Plane distribution.

## Contracts and jobs

Generated HTTP transports derive from checked-in designs. Domain services enforce access control and database constraints.
Request transactions record durable work. Workers execute that work with retries and persisted status.
Do not replace durable effects with detached goroutines.

The route inventory records 639 retired Python application operations. Supplemental authentication redirects also run in Go.
Historical fixtures document compatibility and intentional corrections. They are not a second production backend.
