#!/bin/sh
# shellcheck shell=dash

# This is just a little script that can be downloaded from the internet to
# install pgxman. It detects platform, downloads the zipped release, unzip it
# and copy it to $PGXMAN_DESTIATION_DIR.
# Optinally, you can pass a pgxman file to install extensions.

set -u
set -o noglob

PGXMAN_DOWNLOAD_URL="${PGXMAN_DOWNLOAD_URL:-https://github.com/pgxman/release/releases/latest/download}"
PGXMAN_DESTIATION_DIR="${PGXMAN_DESTIATION_DIR:-/usr/local/bin}"
SUDO="${SUDO:-}"

main() {
    downloader --check
    need_cmd uname
    need_cmd tar
    need_cmd mktemp
    need_cmd chown
    need_cmd cp
    need_cmd git

    install_pgxman
    install_extensions "$@"
}

install_pgxman() {
    get_architecture || return 1
    local _arch="$RETVAL"

    local _targz="pgxman_${_arch}.tar.gz"
    local _url="${PGXMAN_DOWNLOAD_URL}/${_targz}"
    local _destdir="${PGXMAN_DESTIATION_DIR}"
    local _tmpdir
    _tmpdir="$(mktemp -d -t pgxman-install.XXXXXXXXXX)"

    if [ "${SUDO}" = "false" ]; then
        SUDO=
    else
        SUDO=sudo
    fi
    if [ "$(id -u)" -eq 0 ]; then
        SUDO=
    fi

    say "Installing PGXMan for ${_arch}..."
    ensure downloader "${_url}" "${_tmpdir}/${_targz}"
    ensure tar -xf "${_tmpdir}/${_targz}" --directory "${_tmpdir}"
    [ -n "${SUDO}" ] && ensure ${SUDO} chown -R "$(stat -c "%U:%G" "${_destdir}")" "${_tmpdir}/bin"
    ensure ${SUDO} cp -r "${_tmpdir}/bin/." "${_destdir}/"
}

install_extensions() {
    ensure pgxman update
    if [ "$#" -ne "0" ]; then
        for _file in "$@"; do
            echo "Installing extensions from ${_file}..."
            ensure pgxman install --file "$_file" --yes || exit 1
        done
    fi
}

get_architecture() {
    local _ostype _cputype _arch
    _ostype="$(uname -s)"
    _cputype="$(uname -m)"

    case "$_ostype" in
    Linux)
        _ostype=linux
        ;;
    Darwin)
        _ostype=darwin
        ;;
    *)
        err "unrecognized OS type: $_ostype"
        ;;
    esac

    case "$_cputype" in
    i386 | i486 | i686 | i786 | x86)
        _cputype=386
        ;;

    xscale | arm | armv6l | armv7l | armv8l)
        _cputype=armv6
        ;;

    aarch64 | arm64)
        _cputype=arm64
        ;;

    x86_64 | x86-64 | x64 | amd64)
        _cputype=amd64
        ;;

    *)
        err "unknown CPU type: $_cputype"
        ;;
    esac

    _arch="${_ostype}_${_cputype}"
    RETVAL="$_arch"
}

ensure() {
    if ! "$@"; then err "command failed: $*"; fi
}

downloader() {
    local _dld
    if check_cmd curl; then
        _dld=curl
    elif check_cmd wget; then
        _dld=wget
    else
        _dld='curl or wget' # to be used in error message of need_cmd
    fi

    if [ "$1" = --check ]; then
        need_cmd "$_dld"
    elif [ "$_dld" = curl ]; then
        if [ -z "$2" ]; then
            curl --silent --show-error --fail --location "$1"
        else
            curl --silent --show-error --fail --location "$1" --output "$2"
        fi
    elif [ "$_dld" = wget ]; then
        if [ -z "$2" ]; then
            wget "$1"
        else
            wget "$1" -O "$2"
        fi
    else
        err "Unknown downloader" # should not reach here
    fi
}

need_cmd() {
    if ! check_cmd "$1"; then
        err "need '$1' (command not found)"
    fi
}

check_cmd() {
    command -v "$1" >/dev/null 2>&1
}

say() {
    printf 'pgxman-install: %s\n' "$1"
}

err() {
    say "$1" >&2
    exit 1
}

main "$@" || exit 1
