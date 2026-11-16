#!/bin/sh
# Downloads a release of norn and norn-client from GitHub, checks it against
# the checksums of the release and installs both binaries.
#
#   curl -fsSL https://raw.githubusercontent.com/nornforge/norn/main/scripts/install.sh | sh
#   sh install.sh --version v0.2.0 --prefix "$HOME/.local"
#
# Options:
#   --version <tag>   the release to install, e.g. v0.2.0 (default: the latest release)
#   --prefix <dir>    install into <dir>/bin (default: /usr/local)
#   --no-completion   do not install the bash completion
#
# Installing into a directory that is not writable uses sudo.

set -eu

repo="nornforge/norn"
version=""
prefix="/usr/local"
completion=1

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

while [ $# -gt 0 ]; do
	case "$1" in
	--version)
		[ $# -ge 2 ] || fail "--version needs a value"
		version="$2"
		shift 2
		;;
	--prefix)
		[ $# -ge 2 ] || fail "--prefix needs a value"
		prefix="$2"
		shift 2
		;;
	--no-completion)
		completion=0
		shift
		;;
	-h | --help)
		if [ -f "$0" ]; then sed -n "2,13p" "$0" | sed "s/^# \{0,1\}//"; else echo "see https://github.com/$repo/blob/main/scripts/install.sh"; fi
		exit 0
		;;
	*)
		fail "unknown option: $1"
		;;
	esac
done

[ "$(uname -s)" = "Linux" ] || fail "releases exist for Linux only, not for $(uname -s)"
case "$(uname -m)" in
x86_64 | amd64) arch="amd64" ;;
aarch64 | arm64) arch="arm64" ;;
*) fail "no release for the architecture $(uname -m)" ;;
esac

for tool in curl tar sha256sum; do
	command -v "$tool" >/dev/null 2>&1 || fail "$tool is required"
done

if [ -z "$version" ]; then
	# The latest release redirects to its tag, which avoids the rate limit of the API
	latest=$(curl -fsSLo /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest") ||
		fail "unable to find the latest release"
	version="${latest##*/}"
fi
case "$version" in
v[0-9]*) ;;
[0-9]*) version="v$version" ;;
*) fail "invalid version: $version" ;;
esac
number="${version#v}"

archive="norn_${number}_linux_${arch}.tar.gz"
checksums="norn_${number}_checksums.txt"
url="https://github.com/$repo/releases/download/$version"

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT INT TERM

echo "Downloading norn $version for linux/$arch"
curl -fsSL -o "$workdir/$archive" "$url/$archive" || fail "unable to download $url/$archive"
curl -fsSL -o "$workdir/$checksums" "$url/$checksums" || fail "unable to download $url/$checksums"

(
	cd "$workdir"
	grep "  $archive\$" "$checksums" >"$archive.sha256" || fail "$archive is not listed in $checksums"
	sha256sum --check --quiet "$archive.sha256" || fail "the checksum of $archive does not match"
	tar -xzf "$archive" norn norn-client
)

bindir="$prefix/bin"
sudo=""
if [ "$(id -u)" -ne 0 ]; then
	if ! mkdir -p "$bindir" 2>/dev/null || [ ! -w "$bindir" ]; then
		command -v sudo >/dev/null 2>&1 || fail "$bindir is not writable and sudo is not available"
		sudo="sudo"
	fi
fi

$sudo mkdir -p "$bindir"
$sudo install -m 0755 "$workdir/norn" "$workdir/norn-client" "$bindir/"
echo "Installed norn and norn-client to $bindir"

if [ "$completion" -eq 1 ]; then
	if [ "$prefix" = "/usr/local" ] || [ "$prefix" = "/usr" ]; then
		completiondir="$prefix/share/bash-completion/completions"
	else
		completiondir="${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions"
		sudo="" # the completion of a user installation belongs to the user
	fi
	$sudo mkdir -p "$completiondir"
	for binary in norn norn-client; do
		"$workdir/$binary" completion bash >"$workdir/$binary.bash"
		$sudo install -m 0644 "$workdir/$binary.bash" "$completiondir/$binary"
	done
	echo "Installed the bash completion to $completiondir (active in new shells)"
fi

case ":$PATH:" in
*":$bindir:"*) ;;
*) echo "Note: $bindir is not in your PATH" ;;
esac
