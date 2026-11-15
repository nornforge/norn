# CLAUDE.md

Host side of the Norn relay controller: a Go library for the serial protocol, a CLI that talks to the
firmware over USB serial, a REST server in front of it, and a REST client. The firmware lives in the
separate `firmware` repo (`github.com/nornforge/firmware`), which imports `pkg/norn` from here.

## Layout

- `pkg/norn/` – the serial protocol: framing, `Command`, `Response`, command types. Imported by the firmware,
  so it must keep building with TinyGo: standard library only, nothing that needs an OS.
- `cmd/cli/` – `norn`, the CLI talking to the device over a serial port (`on`, `off`, `get`, `boot`, `version`,
  `list`, `server`)
- `cmd/client/` – `norn-client`, the CLI talking to a running `norn server` over REST
- `server/` – the REST server (`/api/v1/relay/{index}/value`, `/api/v2/info`)
- `version/` – program version, injected at build time
- `scripts/systemd/` – unit file shipped in the release archive
- `.goreleaser.yml` – release build (linux amd64/arm64, both binaries)
- `.devcontainer/` – dev container (Go, GoReleaser pinned in `postCreateCommand.sh`)

## Commands

```sh
make all            # build norn and norn-client
go test ./...       # tests
golangci-lint run   # lint (ships with the dev container image)
```

The version is injected via `-ldflags "-X github.com/nornforge/norn/version.VersionInput=<version>"`.

## Conventions

- When bumping Go, change the version in `.devcontainer/devcontainer.json` and in `.github/workflows/*.yml`
  together, so the dev container and CI build with the same toolchain.
- A change to the protocol in `pkg/norn` affects the firmware. Add new command types at the end, never
  renumber existing ones, and update the firmware repo to the new norn release afterwards.
- A connected Pico shows up as `/dev/ttyACM0` inside the dev container; access needs `sudo`
  (e.g. `sudo ./norn version`).
- Commit messages follow Conventional Commits (`feat:`, `fix:`, `ci:` …); the release changelog is generated from them.
