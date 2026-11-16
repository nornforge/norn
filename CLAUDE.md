# CLAUDE.md

Host side of the Norn relay controller: a Go library for the serial protocol, a CLI that talks to the
firmware over USB serial, a REST server in front of it, and a REST client. The firmware lives in the
separate `firmware` repo (`github.com/nornforge/firmware`), which imports `pkg/norn` from here.

## Layout

- `pkg/norn/` – the serial protocol: framing, `Command`, `Response`, command types. Imported by the firmware,
  so it must keep building with TinyGo: standard library only, nothing that needs an OS. Its tests must run
  with TinyGo too, so they don't use testify (it panics under TinyGo); `require_test.go` provides the checks.
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
tinygo test ./pkg/norn                               # pkg/norn tests with TinyGo, as the firmware builds it
tinygo test -c -o /tmp/n.elf -target=pico ./pkg/norn # build pkg/norn for the Pico
golangci-lint run   # lint (ships with the dev container image)
```

The version is injected via `-ldflags "-X github.com/nornforge/norn/version.VersionInput=<version>"`.

## Conventions

- The dev container (`.devcontainer/devcontainer.json`) and the workflows (`.github/workflows/*.yml`) each
  name a Go version. They may differ for a while, but aim for the same one: when bumping Go in one place,
  check the other.
- TinyGo (dev container and the `tinygo` job in `go.yml`) uses the same version as the firmware repository;
  bump it together with the firmware.
- A change to the protocol in `pkg/norn` affects the firmware. Add new command types at the end, never
  renumber existing ones, and update the firmware repo to the new norn release afterwards.
- A connected Pico shows up as `/dev/ttyACM0` inside the dev container; access needs `sudo`
  (e.g. `sudo ./norn version`).
- Commit messages follow Conventional Commits (`feat:`, `fix:`, `ci:` …); the release changelog is generated from them.
