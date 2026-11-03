---
description: Guidelines for bash commands and tooling in the monorepo
applyTo: "**/*.sh"
---

# Bash & Tooling Instructions

This document outlines the standard tools and commands used in this monorepo.

## Package Manager

We use **Bun** for package management.

- **Do not use `npm` or `yarn`.**
- Lockfile: `bun.lock`
- Workspace configuration: `package.json#workspaces`

### Common Commands

- Install dependencies: `bun install --frozen-lockfile`
- Run a script in a specific package: `bun run --filter <package_name> <script>`
- Run a script in all packages: `bun run --filter "*" <script>`

## Monorepo Tooling

We use **Turbo** for build system orchestration.

- Configuration: `turbo.json`

## Project Structure

- `apps/`: Contains application services (admin, api, live, proxy, space, web).
- `packages/`: Contains shared packages and libraries.
- `deployments/`: Deployment configurations.

## Running Tests

- To run tests in a specific package (e.g., codemods):
  ```bash
  cd packages/codemods
  bun run test
  ```
- Or from root:
  ```bash
  bun run --filter @plane/codemods test
  ```

## Docker

- Local development uses `docker-compose-local.yml`.
- Production/Staging uses `docker-compose.yml`.
