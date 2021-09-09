#!/bin/bash

function disable_systemd {
    systemctl disable circonus-unified-agent
    rm -f $1
}

function disable_update_rcd {
    update-rc.d -f circonus-unified-agent remove
    rm -f /etc/init.d/circonus-unified-agent
}

function disable_chkconfig {
    chkconfig --del circonus-unified-agent
    rm -f /etc/init.d/circonus-unified-agent
}

if [[ -f /etc/redhat-release ]] || [[ -f /etc/SuSE-release ]]; then
    # RHEL-variant logic
    if [[ "$1" = "0" ]]; then
        if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
            disable_systemd /usr/lib/systemd/system/circonus-unified-agent.service
        else
            # Assuming sysv
            disable_chkconfig
        fi
    fi
elif [[ -f /etc/os-release ]]; then
    source /etc/os-release
    if [[ "$ID" = "amzn" ]] && [[ "$1" = "0" ]]; then
        if [[ "$NAME" = "Amazon Linux" ]]; then
            # Amazon Linux 2+ logic
            disable_systemd /usr/lib/systemd/system/circonus-unified-agent.service
        elif [[ "$NAME" = "Amazon Linux AMI" ]]; then
            # Amazon Linux logic
            disable_chkconfig
        fi
    elif [[ "$NAME" = "Solus" ]]; then
        disable_systemd /usr/lib/systemd/system/circonus-unified-agent.service
    elif [[ "$NAME" = "SLES" ]]; then
        # SuSE
        disable_systemd /usr/lib/systemd/system/circonus-unified-agent.service
    fi
fi
