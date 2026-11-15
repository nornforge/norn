#!/bin/sh
set -e

GORELEASER_VERSION="2.20.2"
# SHA-256 of goreleaser_Linux_<x86_64|arm64>.tar.gz, update together with the version:
#   gh api repos/goreleaser/goreleaser/releases/tags/v${GORELEASER_VERSION} -q '.assets[] | "\(.name) \(.digest)"'
GORELEASER_SHA256_AMD64="f3d24969a7389a46613875711d15a96d8f6ef1553d84c6ace09d69ac1e7d44c0"
GORELEASER_SHA256_ARM64="f0fa989c3531348e8f975aac647eb187a09046f8a34c54e0b79bed66c84b7f96"

COBRA_CLI_VERSION="1.3.0"

# The volume holding the GitHub CLI login is created owned by root
sudo chown -R "$(id -u):$(id -g)" "${HOME}/.config"

# print the Go version and the version of the linter the image ships
go version
golangci-lint --version

# Install Bash Autocompletion and helpers
sudo apt-get update
sudo apt-get install -q -y bash-completion \
                           jq \
                           wget

ARCH=$(dpkg --print-architecture)
case "${ARCH}" in
    amd64)
        GORELEASER_SHA256="${GORELEASER_SHA256_AMD64}"
        GORELEASER_ARCH="x86_64"
        ;;
    arm64)
        GORELEASER_SHA256="${GORELEASER_SHA256_ARM64}"
        GORELEASER_ARCH="arm64"
        ;;
    *) echo "No checksums known for architecture ${ARCH}" >&2; exit 1 ;;
esac
DL_DIR=$(mktemp -d)

# Install GoReleaser
wget -q -O "${DL_DIR}/goreleaser.tar.gz" \
    "https://github.com/goreleaser/goreleaser/releases/download/v${GORELEASER_VERSION}/goreleaser_Linux_${GORELEASER_ARCH}.tar.gz"
echo "${GORELEASER_SHA256}  ${DL_DIR}/goreleaser.tar.gz" | sha256sum -c -
tar -xzf "${DL_DIR}/goreleaser.tar.gz" -C "${DL_DIR}" goreleaser
sudo install -m 0755 "${DL_DIR}/goreleaser" /usr/local/bin/goreleaser
goreleaser --version

rm -rf "${DL_DIR}"

# Generator for new Cobra commands
go install "github.com/spf13/cobra-cli@v${COBRA_CLI_VERSION}"
