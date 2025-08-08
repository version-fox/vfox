#!/bin/bash

# Error on error
set -e

# ================================================================ #
# Logging
# ================================================================ #

# Log to stderr
echoerr() {
  echo "$@" 1>&2
}

# Log prefix
log_prefix() {
  echo "$0"
}

# Log Priority: 0=emerg, 1=alert, 2=crit, 3=err, 4=warning, 5=notice, 6=info, 7=debug
_logp=6

log_set_priority() {
  _logp="$1"
}

log_priority() {
  if test -z "$1"; then
    echo "$_logp"
    return
  fi
  [ "$1" -le "$_logp" ]
}

log_tag() {
  case $1 in
    0) echo "emerg" ;;
    1) echo "alert" ;;
    2) echo "crit" ;;
    3) echo "err" ;;
    4) echo "warning" ;;
    5) echo "notice" ;;
    6) echo "info" ;;
    7) echo "debug" ;;
    *) echo "$1" ;;
  esac
}
log_debug() {
  log_priority 7 || return 0
  echoerr "$(log_prefix)" "$(log_tag 7)" "$@"
}
log_info() {
  log_priority 6 || return 0
  echoerr "$(log_prefix)" "$(log_tag 6)" "$@"
}
log_err() {
  log_priority 3 || return 0
  echoerr "$(log_prefix)" "$(log_tag 3)" "$@"
}
log_crit() {
  log_priority 2 || return 0
  echoerr "$(log_prefix)" "$(log_tag 2)" "$@"
}

# ================================================================ #
# Helper Functions
# ================================================================ #

# Check if a command exists
is_command() { command -v "${1:?}" >/dev/null 2>&1; }

# Extract archive file
untar() {
  tarball=$1
  case "${tarball}" in
    *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" ;;
    *.tar) tar --no-same-owner -xf "${tarball}" ;;
    *.zip) unzip "${tarball}" ;;
    *)
      log_err "untar unknown archive format for ${tarball}"
      return 1
      ;;
  esac
}

# Download file using curl
http_download_curl() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    code=$(curl -w '%{http_code}' -sL -o "$local_file" "$source_url")
  else
    code=$(curl -w '%{http_code}' -sL -H "$header" -o "$local_file" "$source_url")
  fi
  if [ "$code" != "200" ]; then
    log_debug "http_download_curl received HTTP status $code"
    return 1
  fi
  return 0
}

# Download file using wget
http_download_wget() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    wget -q -O "$local_file" "$source_url"
  else
    wget -q --header "$header" -O "$local_file" "$source_url"
  fi
}

# Download file using curl or wget
http_download() {
  log_debug "http_download $2"
  if is_command curl; then
    http_download_curl "$@"
    return
  elif is_command wget; then
    http_download_wget "$@"
    return
  fi
  log_crit "http_download unable to find wget or curl"
  return 1
}

# Copy file from url
http_copy() {
  tmp=$(mktemp)
  http_download "${tmp}" "$1" "$2" || return 1
  body=$(cat "$tmp")
  rm -f "${tmp}"
  echo "$body"
}

# Get the version of github release
github_release() {
  owner_repo=$1
  version=$2
  test -z "$version" && version="latest"
  giturl="https://github.com/${owner_repo}/releases/${version}"
  json=$(http_copy "$giturl" "Accept:application/json")
  test -z "$json" && return 1
  version=$(echo "$json" | tr -s '\n' ' ' | sed 's/.*"tag_name":"//' | sed 's/".*//')
  test -z "$version" && return 1
  echo "$version"
}

# Get hash of a file using sha-256
hash_sha256() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha256sum; then
    hash=$(gsha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha256sum; then
    hash=$(sha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 256 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl -dst openssl dgst -sha256 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f a
  else
    log_crit "hash_sha256 unable to find command to compute sha-256 hash"
    return 1
  fi
}

# Verify sha-256 hash of a file against a checksum file
hash_sha256_verify() {
  TARGET=$1
  checksums=$2
  if [ -z "$checksums" ]; then
    log_err "hash_sha256_verify checksum file not specified in arg2"
    return 1
  fi
  BASENAME=${TARGET##*/}
  want=$(grep "${BASENAME}" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)
  if [ -z "$want" ]; then
    log_err "hash_sha256_verify unable to find checksum for '${TARGET}' in '${checksums}'"
    return 1
  fi
  got=$(hash_sha256 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_sha256_verify checksum for '$TARGET' did not verify ${want} vs $got"
    return 1
  fi
}

# ================================================================ #
# OS Detection
# ================================================================ #

uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    darwin*) os="macos" ;;
  esac
  echo "$os"
}

uname_arch() {
  arch=$(uname -m)
  case "$arch" in
    arm64) arch="aarch64" ;;
    loongarch64) arch="loong64" ;;
  esac
  echo ${arch}
}

# ================================================================ #
# Main
# ================================================================ #

OWNER=version-fox
REPO=vfox
OS=$(uname_os)
ARCH=$(uname_arch)
PLATFORM="${OS}/${ARCH}"
PREFIX="${OWNER}/${REPO}"
log_prefix() {
	echo "$PREFIX"
}

usage() {
  this=$1
  cat <<EOF
Usage: ${this} [-b] bindir [-d]
  -b bindir   Specify the directory to install the binary (default: \${HOME}/.local/bin)
  -d          Enable debug logging
  -h          Show this help message
EOF
  exit 2
}

parse_args() {
  BINDIR=${BINDIR:-${HOME}/.local/bin}
  while getopts "b:dh" arg; do
    case "$arg" in
      b) BINDIR="$OPTARG" ;;
      d) log_set_priority 10 ;;
      h | \?) usage "$0" ;;
    esac
  done
  shift $((OPTIND - 1))
  TAG=$1
}

tag_to_version() {
  if [ -z "${TAG}" ]; then
    log_info "checking GitHub for latest tag"
  else
    log_info "checking GitHub for tag '${TAG}'"
  fi
  REALTAG=$(github_release "$OWNER/$REPO" "${TAG}") && true
  if test -z "$REALTAG"; then
    log_crit "unable to find '${TAG}' - use 'latest' or see https://github.com/${PREFIX}/releases for details"
    exit 1
  fi
  # if version starts with 'v', remove it
  TAG="$REALTAG"
  VERSION=${TAG#v}
}

get_binaries() {
  case "$PLATFORM" in
    linux/i386) BINARIES="vfox" ;;
    linux/x86_64) BINARIES="vfox" ;;
    linux/aarch64) BINARIES="vfox" ;;
    linux/armv7) BINARIES="vfox" ;;
    linux/loong64) BINARIES="vfox" ;;
    macos/aarch64) BINARIES="vfox" ;;
    macos/x86_64) BINARIES="vfox" ;;
    windows/i386) BINARIES="vfox.exe" ;;
    windows/x86_64) BINARIES="vfox.exe" ;;
    windows/aarch64) BINARIES="vfox.exe" ;;
    *)
      log_crit "platform $PLATFORM is not supported."
      exit 1
      ;;
  esac
}

execute() {
  tmpdir=$(mktemp -d)
  log_debug "downloading files into ${tmpdir}"
  http_download "${tmpdir}/${TARBALL}" "${TARBALL_URL}"
  http_download "${tmpdir}/${CHECKSUM}" "${CHECKSUM_URL}"
  hash_sha256_verify "${tmpdir}/${TARBALL}" "${tmpdir}/${CHECKSUM}"
  srcdir="${tmpdir}/${PACKAGE_NAME}"
  (cd "${tmpdir}" && untar "${TARBALL}" )
  test ! -d "${BINDIR}" && install -d "${BINDIR}"
  for binexe in $BINARIES; do
    install "${srcdir}/${binexe}" "${BINDIR}/"
    log_info "installed ${BINDIR}/${binexe}"
  done
  rm -rf "${tmpdir}"
}

parse_args "$@"

get_binaries

tag_to_version

log_info "found version: ${VERSION} for ${TAG}/${OS}/${ARCH}"

GITHUB_DOWNLOAD=https://github.com/${OWNER}/${REPO}/releases/download
BINARY=vfox
PACKAGE_NAME=${BINARY}_${VERSION}_${OS}_${ARCH}
FORMAT=tar.gz
TARBALL=${PACKAGE_NAME}.${FORMAT}
TARBALL_URL=${GITHUB_DOWNLOAD}/${TAG}/${TARBALL}
CHECKSUM=checksums.txt
CHECKSUM_URL=${GITHUB_DOWNLOAD}/${TAG}/${CHECKSUM}

execute
