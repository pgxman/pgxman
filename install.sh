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
    need_cmd echo
    need_cmd cat

    install_pgxman
    say_success
    pgxman doctor

    install_extensions "$@"
}

install_pgxman() {
    get_architecture || return 1

    case "$OS_TYPE" in
    linux)
        install_pgxman_linux "$CPU_TYPE"
        ;;

    darwin)
        install_pgxman_darwin
        ;;

    *)
        err "unsupported OS: $OS_TYPE"
        ;;
    esac
}

install_extensions() {
    if [ "$#" -ne "0" ]; then
        for _file in "$@"; do
            echo "Installing extensions from ${_file}..."
            ensure pgxman bundle --file "$_file" --yes || exit 1
        done
    fi
}

get_architecture() {
    local _ostype _cputype
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

    CPU_TYPE="$_cputype"
    OS_TYPE="$_ostype"
}

install_pgxman_linux() {
    need_cmd apt

    local _arch="$1"
    local _url="${PGXMAN_DOWNLOAD_URL}/pgxman_linux_${_arch}.deb"
    local _file="/tmp/pgxman_linux_${_arch}.deb"

    SUDO=""
    if [ "$(id -u)" != "0" ]; then
        if
            which sudo >/dev/null 2>&1
        then
            SUDO="sudo"
        else
            echo "Sudo not found. You will need to run this script as root."
            exit
        fi
    fi

    echo "Installing pgxman for Linux ${_arch}..."
    ensure downloader "$_url" "$_file"
    ensure ${SUDO} apt update
    ensure ${SUDO} apt install -y "$_file"
}

install_pgxman_darwin() {
    need_cmd brew

    if brew ls pgxman >/dev/null 2>&1; then
        echo "Upgrading pgxman for macOS..."
        ensure brew upgrade pgxman/tap/pgxman
    else
        echo "Installing pgxman for macOS..."
        ensure brew install pgxman/tap/pgxman
    fi
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

say_success() {
    cat <<EOS
@@@@@@@@    @@@@@@@@ @@@      @@@  @@@@@@ @@@@@   @@@@@@    @@@@@@
@@@   @@@  @@@    @@@  @@@@  @@@@   @@  @@@@  @@@ @@@  @@@  @@@  @@@
@@      @@ @@@     @@    @@@@@@     @@   @@    @@ @@@@@@@@  @@    @@
@@@    @@@ @@@     @@   @@@  @@@    @@   @@    @@ @@    @@  @@    @@
@@@@@@@@@   @@@@@@@@@ @@@      @@@  @@   @@    @@ @@    @@  @@    @@
@@                 @@
@@          @@@@@@@@@

👏🎉 pgxman successfully installed.
If this is your first time using pgxman, check out our docs at https://docs.pgxman.com/
EOS
}

main "$@" || exit 1
