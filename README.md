# Vector

Open-source project management with a Go backend.

Vector brings work items, projects, cycles, modules, pages, and team collaboration into one self-hosted application.

Vector is an independent derivative of [Plane](https://github.com/makeplane/plane). It is not an official Plane release or service.
The application API, authentication, database bootstrap, and background jobs run in Go. The current React frontend derives from Plane.

## Run locally

Install Git, Go 1.27.3, Node.js 22.18 or later, Corepack, and Docker with Compose and Buildx.

```sh
git clone https://github.com/CaliLuke/vector.git
cd vector
./setup.sh
pnpm dev
```

Open [Vector](http://localhost:8000) and [instance administration](http://localhost:8000/god-mode/).
Use the proxy on port 8000 for browser access. See the [development guide](docs/development.md) for configuration and tests.

## Documentation

- [Development](docs/development.md)
- [Self-hosting](docs/self-hosting.md)
- [Architecture](docs/architecture.md)
- [Privacy and external services](docs/privacy.md)
- [Contributing](CONTRIBUTING.md)
- [Security reports](SECURITY.md)
- [Attribution and license](NOTICE.md)

Report bugs and request features in [Vector issues](https://github.com/CaliLuke/vector/issues).
Vector has no hosted cloud service or published container release yet. Build this repository to run this version.

## License and origin

Vector is licensed under [GNU AGPL v3](LICENSE.txt), with the existing `AGPL-3.0-only` source notices retained.
Copyright notices for Plane Software, Inc. and other contributors remain in the source.
[NOTICE.md](NOTICE.md) describes the upstream work and the changes made for Vector.

This repository starts with one commit after the Go backend cutover. Earlier upstream history remains available in the Plane repository.
