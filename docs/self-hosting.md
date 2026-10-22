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
