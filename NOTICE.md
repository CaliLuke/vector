# Vector: attribution and modification notice

Vector is an independent project maintained by [Luca Candela](https://github.com/CaliLuke).
It derives from [Plane](https://github.com/makeplane/plane), created by Plane Software, Inc. and its contributors.
Vector is not affiliated with, endorsed by, or distributed by Plane Software, Inc.
The Plane name identifies the upstream project and the origin of retained code.

## Copyright and license

Copyright (c) 2023-present Plane Software, Inc. and contributors.
Vector modifications: copyright (c) 2026 Luca Candela and contributors.

The project remains under GNU Affero General Public License version 3 only (`AGPL-3.0-only`).
See [LICENSE.txt](LICENSE.txt) for the complete license and warranty disclaimer.
Existing source notices and third-party licenses remain intact. Dependencies retain their respective licenses.

## Modifications as of October 22, 2026

Vector replaces the application backend with Go, including API handlers, authentication, schema bootstrap, and durable background jobs.
The production proxy routes application requests to Go. The browser applications and collaboration service retain substantial upstream code.
Vector also changes product branding, documentation, development tooling, and deployment configuration.

The initial Vector commit is a snapshot after the backend cutover. It does not retain the upstream Git history.
The [Plane repository](https://github.com/makeplane/plane) retains upstream history and contributor records.
Historical compatibility fixtures retain upstream names where they describe the original protocol or data model.

## Source access

The source is available at https://github.com/CaliLuke/vector.
If you modify and operate this software over a network, provide users access to the corresponding source under the AGPL terms.
