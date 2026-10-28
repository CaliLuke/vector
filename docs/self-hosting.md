# Self-host Vector

Build Vector from this repository. Upstream Plane images do not contain the Vector backend or branding.
No public Vector container release is available yet.

## Build and start with Compose

Install Docker with Compose and Buildx. Prepare the environment files from the repository root:

```sh
./setup.sh
```

Before exposing the instance, replace all example passwords and storage credentials in `.env` and `apps/go-api/.env`.
Configure the public URLs, allowed origins, SMTP, database, and S3 endpoint for your deployment.
Keep `SECRET_KEY` stable across restarts. Store it with your deployment secrets.

Start the source-built production stack:

```sh
docker compose -f docker-compose.yml up -d --build
```

The Go bootstrap container applies schema changes before the API starts.
The stack includes PostgreSQL, Redis-compatible caching, S3-compatible storage, frontend services, collaboration, and Caddy.
The API process also runs native background jobs.

Check service status and logs:

```sh
docker compose -f docker-compose.yml ps
docker compose -f docker-compose.yml logs --tail=100 go-migrator go-api proxy
```

Use the public proxy URL for the web application, `/god-mode/` for administration, and `/spaces/` for public projects.
Configure HTTPS before accepting users. Set S3 browser-upload origins to the same public application origin.

## Logs, traces, and metrics

Vector has native OpenTelemetry exporters for API logs, request traces, and instance usage metrics.
Configure each requested signal: collecting container stdout alone does not configure the native exporters,
provide request traces, or populate the destination shown in God mode.

For the production Compose stack, add these settings to `apps/go-api/.env`:

```dotenv
# Native structured logs and HTTP request traces
OTEL_SDK_DISABLED=false
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_SERVICE_NAME=vector-api
OTEL_RESOURCE_ATTRIBUTES=application.name=vector,service.namespace=vector

# Native instance and workspace usage metrics: a separate exporter
OTLP_ENDPOINT=http://otel-collector:4318
OTLP_METRICS_PROTOCOL=http
SERVICE_NAME=vector-api
```

Replace `otel-collector` with your collector's hostname as reachable from the API container.
The example assumes a private OTLP/HTTP receiver on port 4318 that accepts logs, traces, and metrics.
It does not install a collector. For SigNoz, point these settings at its collector's OTLP/HTTP receiver.
Use a base URL without `/v1/logs`, `/v1/traces`, or `/v1/metrics`; the exporters add the signal paths.
Use HTTPS when traffic crosses an untrusted network, and keep any collector authentication headers in
secret configuration rather than committed files.

The two protocol settings deliberately differ: `OTEL_EXPORTER_OTLP_PROTOCOL` uses `http/protobuf`,
while `OTLP_METRICS_PROTOCOL` uses `http`. Instance metrics default to gRPC if the latter is omitted;
pointing that default at an HTTP receiver will not work. `OTEL_SERVICE_NAME` names the logs/traces service,
while `SERVICE_NAME` names the usage-metrics service. Set both consistently.

Apply environment changes by recreating the API container:

```sh
docker compose -f docker-compose.yml up -d --no-deps go-api
```

This briefly restarts the API. Restarting a container without recreating it does not load changed Compose
configuration. For other deployment systems, update the API environment in their source of truth and
roll out the API workload; changing a ConfigMap alone does not update a running process's environment.

### Check configuration and delivery

In `/god-mode/general/`, the Telemetry section reports these independent settings:

| Entry            | Required configuration                                                                               |
| ---------------- | ---------------------------------------------------------------------------------------------------- |
| Logs and traces  | `OTEL_EXPORTER_OTLP_ENDPOINT` set and `OTEL_SDK_DISABLED` not `true`                                 |
| Usage metrics    | `OTLP_ENDPOINT` set, valid `OTLP_METRICS_PROTOCOL`, and the instance's usage-metrics setting enabled |
| Analytics events | A separately configured `POSTHOG_HOST` and `POSTHOG_API_KEY`                                         |

The page reports configuration, not delivery health. Enabled means an exporter is configured;
it does not confirm that the collector is reachable or accepting data. No destination configured
means that exporter's endpoint is absent, even if an external collector is already forwarding stdout.

Verify all signals you intend to enable:

1. Make a fresh request to `/api/instances/`. In your telemetry backend, confirm a server span and
   structured request logs with matching trace and span IDs under the configured service name.
2. Check for fresh `plane_instance_*` and, when workspaces exist, `plane_workspace_*` metric samples
   from that service. These are application usage counts, not HTTP latency histograms or container
   memory metrics. Infrastructure monitoring must collect resource metrics separately.
3. Instance usage metrics have a default collection interval of six hours. Set
   `METRICS_PUSH_INTERVAL_MINUTES` before startup to choose another interval. An existing durable
   schedule can retain its pending timer, so an environment change does not guarantee an immediate push.
4. For an immediate collection after configuring a running instance, the existing installation command
   admits a durable metrics job:

   ```sh
   docker compose -f docker-compose.yml exec go-api /go-bootstrap install
   ```

   This performs idempotent installation checks and admits background work; it is not a read-only
   diagnostic. Wait for the job to complete and verify a fresh sample in the telemetry backend.

5. If your collector also tails API stdout, exclude the API container from that collection once native
   logging is verified. Otherwise each request log can arrive twice. Supporting containers can continue
   using stdout collection. Recheck one fresh request to confirm there are no duplicate copies.

Keep optional external analytics separate from local observability. PostHog is not required to send
logs, traces, or usage metrics to SigNoz. Do not enable analytics merely to make every status row Enabled.
See [privacy and external services](privacy.md) for the data-sharing boundary.

## Data and upgrades

Back up PostgreSQL, uploaded objects, and deployment secrets before upgrades.
Test restore procedures on a separate instance. Preserve the Compose volumes when restarting or updating containers.
Pin the Git revision that you deploy. Build that revision and check its release notes before applying schema changes.

An existing Plane database requires a separate migration rehearsal with a backup.
The completed development cutover does not certify every upstream release or custom installation.

## Source and privacy

Provide users a link to the source revision that runs your service, including local modifications required by the AGPL.
Read [privacy and external services](privacy.md) before enabling optional integrations or telemetry.
The instance operator supplies its own privacy notice, retention policy, and support contact.

The supported deployment path is the source-built Compose stack described above.
