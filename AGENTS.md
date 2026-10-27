Copyright (c) 2026 Digital Asset (Switzerland) GmbH and/or its affiliates. All rights reserved.
SPDX-License-Identifier: Apache-2.0

# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project overview

`darsyncer` is a Go CLI/service from Digital Asset. It reads DAR files (Daml archives), then uploads and vets them on a Canton participant over gRPC. It can authenticate with a static JWT or OAuth client credentials.

- Module: `github.com/digital-asset/darsyncer`
- Go version: see `go.mod` (currently `go 1.26.6`)
- User docs: https://docs.canton.network/sdks-tools/development-tools/darsyncer

## Layout

| Path | Purpose |
|------|---------|
| `cmd/darsyncer/main.go` | CLI entrypoint (urfave/cli v3): flags, auth selection, transport security |
| `cmd/darsyncer/internal/tokensources` | Token sources for the CLI (e.g. static token from file) |
| `cmd/darsyncer/internal/fstoken` | File-backed token handling |
| `internal/darsyncer` | Core service: `Service.Run`, sync loop (`sync.go`), DAR package DB (`packagedb.go`), health probe (`probe.go`), TLS policy (`tls.go`) |
| `internal/auth0` | Auth0 helpers |
| `internal/logger` | slog setup and CLI integration |
| `internal/stopwatch` | Timing utilities (has tests) |
| `internal/tokenstore` | Token storage |
| `internal/validator/identities` | Identity validation |
| `bin/get-snapshot-version` | Computes the snapshot version from `VERSION` and git metadata |
| `scripts/build.sh` | Builds multi-arch images with `ko` (`--local` outside CI) |
| `scripts/gha/release.sh` | Promotes a snapshot image to the public registry with `gcrane` |
| `nix/` | Dev shell (go, ko, crane, gcrane, gcloud, openjdk21) |
| `VERSION` | Release version (source of truth) |

## Dev environment

- Use the Nix dev shell in `nix/` so the toolchain matches CI.
- `bin/` must be on `PATH` for the scripts that call `get-snapshot-version`.

## Common commands

```sh
go build ./...
go test -v ./...          # required before submitting changes
go vet ./...
gofmt -l .                # should print nothing
go run ./cmd/darsyncer --help
go run ./cmd/darsyncer --dars ./path/to/dars --just-print
./scripts/build.sh        # local ko image build (needs bin/ on PATH)
```

## CLI behavior (keep in sync when changing flags)

- `--dars` (env `DARS`) is required. It takes a directory of DARs or a single DAR.
- Mode (mutually exclusive, one required):
  - `--just-print`: print the package IDs and exit.
  - `--endpoint` (env `ENDPOINT`), optionally with `--check`, `--wait`, `--retry-delay` (default 15s), or `--probe-endpoint` (default `0.0.0.0:8080`).
- Auth (mutually exclusive, optional):
  - `--token-path` (env `TOKEN_PATH`).
  - OAuth: `--client-id`, `--client-secret`, `--oauth-domain`, `--audience`, `--scope` (env vars use the same names in upper snake case). Audience and scope are space-separated.
- `--transport-security`: `https-with-plaintext-fallback` (default, logs a warning) or `https-only`.

## Conventions

- Every source file starts with the Apache-2.0 Digital Asset copyright header. Copy it from an existing file.
- Log with `log/slog` and the `*Context` variants. Use structured attributes, not string formatting.
- Return errors instead of exiting. Only `main` calls `os.Exit`.
- New features must include tests (see `CONTRIBUTING.md`). Use `testify`.
- Keep packages under `internal/`. Don't add a public API surface.
- Never log secrets such as client secrets or token contents.

## Versioning and release

- Bump `VERSION` for releases.
- Snapshot images are pushed to `europe-docker.pkg.dev/da-images/public-unstable/docker/darsyncer:<snapshot>`.
- `scripts/gha/release.sh` copies the snapshot to `.../public/docker/darsyncer:<VERSION>` and tags it `latest`. It skips this step if the release tag already exists.
- Don't run the release or image push scripts unless explicitly asked.
