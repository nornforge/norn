#!/bin/sh
set -e

GORELEASER_VERSION="2.20.2"
# SHA-256 of goreleaser_Linux_<x86_64|arm64>.tar.gz, update together with the version:
#   gh api repos/goreleaser/goreleaser/releases/tags/v${GORELEASER_VERSION} -q '.assets[] | "\(.name) \(.digest)"'
GORELEASER_SHA256_AMD64="f3d24969a7389a46613875711d15a96d8f6ef1553d84c6ace09d69ac1e7d44c0"
GORELEASER_SHA256_ARM64="f0fa989c3531348e8f975aac647eb187a09046f8a34c54e0b79bed66c84b7f96"

COBRA_CLI_VERSION="1.3.0"

# pkg/norn is built into the firmware with TinyGo; keep the version the same
# as in the firmware repository (dev container and workflows).
TINY_GO_VERSION="0.44.0"
# SHA-256 of tinygo_${TINY_GO_VERSION}_<arch>.deb, update together with the version:
#   gh api repos/tinygo-org/tinygo/releases/tags/v${TINY_GO_VERSION} -q '.assets[] | "\(.name) \(.digest)"'
TINY_GO_SHA256_AMD64="8810b20d88236a1353e35641a6a56df1720a7db97006aacc2c6656a3bb090b74"
TINY_GO_SHA256_ARM64="c2addff1430c7066fc11900e93ee8d6dd14fce343a7b0e1d72da0509a58766ff"

# The volume holding the GitHub CLI login is created owned by root, and so is
# its parent directory if Docker had to create it for the mount
sudo chown "$(id -u):$(id -g)" "${HOME}/.config"
sudo chown -R "$(id -u):$(id -g)" "${HOME}/.config/gh"

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
        TINY_GO_SHA256="${TINY_GO_SHA256_AMD64}"
        ;;
    arm64)
        GORELEASER_SHA256="${GORELEASER_SHA256_ARM64}"
        GORELEASER_ARCH="arm64"
        TINY_GO_SHA256="${TINY_GO_SHA256_ARM64}"
        ;;
    *) echo "No checksums known for architecture ${ARCH}" >&2; exit 1 ;;
esac
DL_DIR=$(mktemp -d)
# Remove the downloads again, also when a step below fails
trap 'rm -rf "${DL_DIR}"' EXIT

# Install GoReleaser
wget -q -O "${DL_DIR}/goreleaser.tar.gz" \
    "https://github.com/goreleaser/goreleaser/releases/download/v${GORELEASER_VERSION}/goreleaser_Linux_${GORELEASER_ARCH}.tar.gz"
echo "${GORELEASER_SHA256}  ${DL_DIR}/goreleaser.tar.gz" | sha256sum -c -
tar -xzf "${DL_DIR}/goreleaser.tar.gz" -C "${DL_DIR}" goreleaser
sudo install -m 0755 "${DL_DIR}/goreleaser" /usr/local/bin/goreleaser
goreleaser --version

# Install TinyGo, to check that pkg/norn still builds for the firmware:
#   tinygo test ./pkg/norn
TINY_GO_DEB="tinygo_${TINY_GO_VERSION}_${ARCH}.deb"
wget -q -O "${DL_DIR}/${TINY_GO_DEB}" \
    "https://github.com/tinygo-org/tinygo/releases/download/v${TINY_GO_VERSION}/${TINY_GO_DEB}"
echo "${TINY_GO_SHA256}  ${DL_DIR}/${TINY_GO_DEB}" | sha256sum -c -
sudo dpkg -i "${DL_DIR}/${TINY_GO_DEB}"
tinygo version

# Generator for new Cobra commands
go install "github.com/spf13/cobra-cli@v${COBRA_CLI_VERSION}"
