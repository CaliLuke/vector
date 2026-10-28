# Privacy and external services

Vector is self-hosted software. The instance operator controls accounts, application data, backups, logs, and retention.
This repository does not provide a hosted service or a universal privacy policy for deployments.

The operator must publish the terms and privacy notice that apply to its instance.
Plane's hosted-service terms and privacy policy do not govern a Vector deployment on behalf of the Vector maintainer.

Optional integrations can send data to configured email, OAuth, AI, webhook, analytics, or storage providers.
Backend instance metrics are disabled unless you configure `OTLP_ENDPOINT`.
Review optional frontend analytics and integration settings before exposing an instance.
Instance admins can view telemetry status and destination servers in God mode's General settings.
This read-only view reports configuration, not successful delivery. Logs and traces use
`OTEL_EXPORTER_OTLP_ENDPOINT` and respect `OTEL_SDK_DISABLED`; usage metrics require
`OTLP_ENDPOINT` and the instance's metrics setting. Analytics events require both
`POSTHOG_HOST` and `POSTHOG_API_KEY`. Destinations show only the server address, without
credentials, paths, query parameters, or fragments.
Local development logs can contain account identifiers and request paths. Keep logs private and redact them before sharing.

For vulnerabilities, use [private security reporting](../SECURITY.md).
For source access, see [the repository](https://github.com/CaliLuke/vector) and [license](../LICENSE.txt).
