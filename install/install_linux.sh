#!/usr/bin/env bash

set -o errtrace
set -o errexit
set -o nounset

# ignore tput errors for terms that do not
# support colors (colors will be blank strings)
set +e
RED=$(tput setaf 1)
GREEN=$(tput setaf 2)
NORMAL=$(tput sgr0)
BOLD=$(tput bold)
set -e

cua_version=""
# amd64 or arm64
# (note, x86_64 will be aliased to amd64)
# (note, aarch64 will be aliased to arm64)
pkg_arch="amd64" # default
pkg_ext=""       # currently: rpm or deb
pkg_cmd=""       # currently: yum or dpkg
pkg_args=""
pkg_file=""
pkg_url=""
cua_api_key=""
cua_api_app=""
cua_api_url=""
cua_conf_file="/opt/circonus/unified-agent/etc/circonus-unified-agent.conf"

usage() {
  printf "%b" "Circonus Unified Agent Install Help

Usage

  ${GREEN}install.sh --key <apikey>${NORMAL}

Options

  --key           Circonus API key/token **${BOLD}REQUIRED${NORMAL}**
  [--app]         Circonus API app name (authorized w/key) Default: circonus-unified-agent
  [--apiurl]      Circonus API URL (e.g. https://api.circonus.com/v2) Default: https://api.circonus.com/v2
  [--arch]        Circonus package architecture (amd64, arm64) Default: return from 'uname -m'
  [--ver]         Install specific version (use semver tag from repository releases - e.g. v0.0.32)
  [--help]        This message

Note: Provide an authorized app for the key or ensure api 
      key/token has adequate privileges (default app state:allow)
"
}

log()  { printf "%b\n" "$*"; }
fail() { printf "${RED}" >&2; log "\nERROR: $*\n" >&2; printf "${NORMAL}" >&2; exit 1; }
pass() { printf "${GREEN}"; log "$*"; printf "${NORMAL}"; }

__parse_parameters() {
    local token=""
    log "Parsing command line parameters"
    while (( $# > 0 )) ; do
        token="$1"
        shift
        case "$token" in
        (--key)
            if [[ -n "${1:-}" ]]; then
                cua_api_key="$1"
                shift
            else
                fail "--key must be followed by an api key."
            fi
            ;;
        (--app)
            if [[ -n "${1:-}" ]]; then
                cua_api_app="$1"
                shift
            else
                fail "--app must be followed by an api app."
            fi
            ;;
        (--apiurl)
            if [[ -n "${1:-}" ]]; then
                cua_api_url="$1"
                shift
            else
                fail "--url must be followed by an api url."
            fi
        (--arch)
            if [[ -n "${1:-}" ]]; then
                pkg_arch="$1"
                shift
            else
                fail "--arch must be followed by an architecture."
            fi
            ;;
        (--ver)
            if [[ -n "${1:-}" ]]; then
                ver="$1"
                [[ $ver =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]] && cua_version=${ver#v} || fail "--ver must be followed by a valid semver (e.g. v0.0.32)."
                shift                
            else
                fail "--ver must be followed by a valid semver (e.g. v0.0.32)."
            fi
            ;;
        (--help)
            usage
            exit 0
            ;;
        (*)
            fail "Unknown parameter: $token"
            ;;
        esac
    done
}

__cua_init() {
    set +o errexit

    # set the package architecture
    # (note, an explicit --arch parameter will override this)
    case "$(uname -m)" in
    (amd64|x86_64)
        pkg_arch="amd64"
        ;;
    (arm64|aarch64)
        pkg_arch="arm64"
        ;;
    (*)
        fail "Unsupported architecture: $os_arch"
        ;;
    esac    
    
    # trigger error if needed commands are not found...
    local cmd_list="cat curl sed uname mkdir systemctl basename"
    local cmd
    for cmd in $cmd_list; do
        type -P $cmd >/dev/null 2>&1 || fail "Unable to find '${cmd}' command. Ensure it is available in PATH '${PATH}' before continuing."
    done

    # detect package installation command
    cmd_list="yum dpkg rpm"
    for cmd in $cmd_list; do
        pkg_cmd=$(type -P $cmd)
        if [[ $? -eq 0 ]]; then
            case "$(basename $pkg_cmd)" in
            (yum)
                pkg_ext=".rpm"
                pkg_args="localinstall -y"
                ;;
            (rpm)
                pkg_ext=".rpm"
                pkg_args="-iv"
                ;;
            (dpkg)
                pkg_ext=".deb"
                pkg_args="--install --force-confold"
                ;;
            esac
            break
        fi
    done

    [[ -n "${pkg_cmd:-}" ]] || fail "Unable to find a package install command ($cmd_list)"

    set -o errexit

    __parse_parameters "$@" 
    [[ -n "${cua_api_key:-}" ]] || fail "Circonus API key is *required*."
}

__make_circonus_dir() {
    local circ_dir="/opt/circonus"

    log "Creating Circonus base directory: ${circ_dir}"
    if [[ ! -d $circ_dir ]]; then
        \mkdir -p $circ_dir
        [[ $? -eq 0 ]] || fail "unable to create ${circ_dir}"
    fi

    log "Changing to ${circ_dir}"
    \cd $circ_dir
    [[ $? -eq 0 ]] || fail "unable to change to ${circ_dir}"
}

__get_cua_package() {
    local pkg="${pkg_file}${pkg_ext}"
    local url="${pkg_url}${pkg}"

    if [[ ! -f $pkg ]]; then
        log "Downloading agent package: ${url}"
        set +o errexit
        \curl -sLO "$url"
        curl_err=$?
        set -o errexit
        [[ $curl_err -eq 0 ]] || fail "unable to download ${url} ($curl_err)"
    fi

    [[ -f $pkg ]] || fail "unable to find ${pkg} in current dir"

    log "Installing: ${pkg_cmd} ${pkg_args} ${pkg}"
    $pkg_cmd $pkg_args $pkg
    [[ $? -eq 0 ]] || fail "installing ${pkg_cmd} ${pkg_args} ${pkg}"
}

__configure_agent() {
    log "Updating configuration: ${cua_conf_file}"

    [[ -f $cua_conf_file ]] || fail "config file (${cua_conf_file}) not found"

    log "\tSetting Circonus API key in configuration"
    \sed -i -e "s/    api_token = \"\"/    api_token = \"${cua_api_key}\"/" $cua_conf_file
    [[ $? -eq 0 ]] || fail "updating ${cua_conf_file} with api key"

    if [[ -n "${cua_api_app}" ]]; then
        log "\tSetting Circonus API app name in configuration"
        \sed -i -e "s/    # api_app = \"circonus-unified-agent\"/    api_app = \"${cua_api_app}\"/" $cua_conf_file
        [[ $? -eq 0 ]] || fail "updating ${cua_conf_file} with api app"
    fi

    if [[ -n "${cua_api_url}" ]]; then
        log "\tSetting Circonus API URL in configuration"
        \sed -i -e "s/    # api_url = \"https://api.circonus.com/\"/    api_url = \"${cua_api_url}\"/" $cua_conf_file
        [[ $? -eq 0 ]] || fail "updating ${cua_conf_file} with api url"
    fi

    log "Restarting circonus-unified-agent service"

    \systemctl restart circonus-unified-agent
    [[ $? -eq 0 ]] || fail "systemctl restart circonus-unified-agent failed"
}

__get_latest_release() {
    local url="https://api.github.com/repos/circonus-labs/circonus-unified-agent/releases/latest"

    set +o errexit
    \curl -sS $url  | grep -Po '"tag_name": "\K.*?(?=")'
    curl_err=$?
    set -o errexit

    [[ $curl_err -eq 0 ]] || fail "unable to get latest release (${curl_err})"
}

cua_install() {
    __cua_init "$@"
    if [[ -z "$cua_version" ]]; then
        log "Getting latest release version from repository"
        tag=$(__get_latest_release)
        cua_version=${tag#v}
    fi

    pkg_file="circonus-unified-agent_${cua_version}_${pkg_arch}"
    pkg_url="https://github.com/circonus-labs/circonus-unified-agent/releases/download/v${cua_version}/"

    log "Installing Circonus Unified Agent v${cua_version} for ${pkg_arch}"

    cua_dir="/opt/circonus/unified-agent"
    [[ -d $cua_dir ]] && fail "${cua_dir} previous installation directory found."

    __make_circonus_dir
    __get_cua_package
    __configure_agent

    echo
    echo
    pass "Circonus Unified Agent v${cua_version} installed"
    echo
    log "Make any additional customization to configuration:"
    log "  ${cua_conf_file}"
    log "and restart agent for changes to take effect."
    echo
    echo
}

#
# no arguments are passed
#
if [[ $# -eq 0 ]]; then
    usage
    exit 0
fi
# short-circuit for help
if [[ "$*" == *--help* ]]; then
    usage
    exit 0
fi

#
# NOTE Ensure sufficient rights to do the install
#
(( UID != 0 )) && {
    printf "\n%b\n\n" "${RED}Must run as root[sudo] -- installing software requires certain permissions.${NORMAL}"
    exit 1
}

cua_install "$@"

# END
