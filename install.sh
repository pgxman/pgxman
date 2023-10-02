#!/bin/sh
# shellcheck shell=dash

# This is just a little script that can be downloaded from the internet to
# install pgxman. It just does platform detection, downloads the package
# and use corresponding package manager to install it.
# Optinally, you can pass a pgxman file to install extensions.

set -u
set -o noglob

PGXMAN_DOWNLOAD_URL="${PGXMAN_DOWNLOAD_URL:-https://github.com/pgxman/release/releases/latest/download}"

main() {
    downloader --check
    need_cmd uname
    need_cmd apt

    install_pgxman
    install_extensions "$@"
}

install_pgxman() {
    get_architecture || return 1

    local _arch="$RETVAL"
    local _url="${PGXMAN_DOWNLOAD_URL}/pgxman_linux_${_arch}.deb"
    local _file="/tmp/pgxman_linux_${_arch}.deb"

    echo "Installing PGXMan for ${_arch}..."
    ensure downloader "$_url" "$_file"
    ensure apt update
    ensure apt install -y "$_file"
}

install_extensions() {
    if [ "$#" -ne "0" ]; then
        for _file in "$@"; do
            echo "Installing extensions from ${_file}..."
            ensure pgxman install --file "$_file" --yes || exit 1
        done
    fi
}

get_architecture() {
    local _cputype
    _cputype="$(uname -m)"

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

    RETVAL="$_cputype"
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
