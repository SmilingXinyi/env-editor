#!/bin/sh
# Install or upgrade the env-editor command for the current user.
#
# The same command does both. It downloads the latest GitHub Release for
# this machine, checks the archive against SHA256SUMS, and moves the binary
# to the user's bin directory (default: ~/.local/bin). An existing binary
# at that path is replaced. The short command ee in that directory is a
# link to the binary. The archive, checksum, and temporary copy are removed
# before the script exits.
#
#   curl -fsSL https://raw.githubusercontent.com/SmilingXinyi/env-editor/main/install.sh | sh
#
# Choose another directory with ENV_EDITOR_BIN:
#
#   curl -fsSL https://raw.githubusercontent.com/SmilingXinyi/env-editor/main/install.sh | ENV_EDITOR_BIN="$HOME/bin" sh

set -eu

repo="SmilingXinyi/env-editor"
name="env-editor"
short="ee"
tmpdir=
stage=

die() {
  printf 'install: %s\n' "$1" >&2
  exit 1
}

cleanup() {
  if [ -n "${tmpdir:-}" ] && [ -d "$tmpdir" ]; then
    rm -rf "$tmpdir"
  fi
  if [ -n "${stage:-}" ]; then
    rm -f "$stage"
  fi
}

trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM
trap cleanup EXIT

need() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

need curl
need tar
need awk
need ln
need readlink

os=$(uname -s)
case "$os" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) die "unsupported operating system: $os" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) die "unsupported architecture: $arch" ;;
esac

if [ -n "${ENV_EDITOR_BIN:-}" ]; then
  bindir=$ENV_EDITOR_BIN
elif [ -n "${HOME:-}" ]; then
  bindir=$HOME/.local/bin
else
  die "HOME is not set; set ENV_EDITOR_BIN to the directory that should hold ${name}"
fi

mkdir -p "$bindir"
bindir=$(cd -- "$bindir" >/dev/null && pwd)
dest=$bindir/$name

if [ -e "$dest" ] && [ ! -f "$dest" ]; then
  die "${dest} exists and is not a file"
fi

latest=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${repo}/releases/latest") || die "could not resolve the latest release"
latest=${latest%/}
tag=${latest##*/}
case "$tag" in
  v*) ;;
  *) die "could not resolve the latest release (got: ${tag})" ;;
esac
version=${tag#v}

asset="${name}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/${repo}/releases/download/${tag}"

tmpdir=$(mktemp -d)

curl -fsSL "${base}/SHA256SUMS" -o "$tmpdir/SHA256SUMS" || die "could not download checksums for ${tag}"
curl -fsSL "${base}/${asset}" -o "$tmpdir/$asset" || die "no archive for ${os}/${arch} in ${tag} (${asset})"

expected=$(awk -v file="$asset" '
  {
    name = $2
    sub(/^\*/, "", name)
    if (name == file) {
      print $1
      exit
    }
  }
' "$tmpdir/SHA256SUMS")
expected=$(printf '%s' "$expected" | tr 'A-F' 'a-f')
ok=$(printf '%s' "$expected" | awk '/^[0-9a-f]{64}$/ { print "yes"; exit }')
[ "$ok" = "yes" ] || die "checksum entry missing for ${asset}"

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmpdir/$asset" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmpdir/$asset" | awk '{ print $1 }')
else
  die "required command not found: sha256sum or shasum"
fi
actual=$(printf '%s' "$actual" | tr 'A-F' 'a-f')
[ "$expected" = "$actual" ] || die "checksum mismatch for ${asset}"

tar -xzf "$tmpdir/$asset" -C "$tmpdir" || die "could not extract ${asset}"
src=$tmpdir/${name}_${version}_${os}_${arch}/$name
[ -f "$src" ] || die "archive does not contain ${name}"

if [ -e "$dest" ]; then
  action=Updated
else
  action=Installed
fi

stage=$bindir/.${name}.new.$$
cp "$src" "$stage" 2>/dev/null || die "cannot write ${dest}"
chmod 755 "$stage" 2>/dev/null || die "cannot make ${dest} executable"
"$stage" --help >/dev/null 2>&1 || die "downloaded binary did not run"
mv -f "$stage" "$dest" 2>/dev/null || die "cannot replace ${dest}"
stage=

link=$bindir/$short
link_ok=0
if [ ! -e "$link" ] && [ ! -L "$link" ]; then
  link_ok=1
else
  target=$(readlink "$link" 2>/dev/null || true)
  case "$target" in
    "$name" | "$dest") link_ok=1 ;;
  esac
fi
if [ "$link_ok" -eq 1 ]; then
  stage=$bindir/.${short}.new.$$
  ln -s "$name" "$stage" 2>/dev/null || die "cannot link ${link}"
  mv -f "$stage" "$link" 2>/dev/null || die "cannot link ${link}"
  stage=
else
  printf 'install: left %s unchanged\n' "$link" >&2
fi

printf '%s %s at %s\n' "$action" "$tag" "$dest"
if [ -L "$link" ]; then
  target=$(readlink "$link" 2>/dev/null || true)
  case "$target" in
    "$name" | "$dest") printf 'Short command: %s\n' "$link" ;;
  esac
fi

case ":$PATH:" in
  *":$bindir:"*) ;;
  *)
    printf "Add the directory to PATH:\n  export PATH=\"%s:\$PATH\"\n" "$bindir"
    ;;
esac

resolved=$(command -v "$name" || true)
if [ -n "$resolved" ] && [ "$resolved" != "$dest" ]; then
  printf 'A different %s is earlier on PATH (%s).\n' "$name" "$resolved" >&2
fi
resolved=$(command -v "$short" || true)
if [ -n "$resolved" ] && [ "$resolved" != "$link" ]; then
  printf 'A different %s is earlier on PATH (%s).\n' "$short" "$resolved" >&2
fi
