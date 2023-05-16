#!/bin/bash

BIN_DIR=/opt/circonus/unified-agent/sbin
SERVICE_DIR=/opt/circonus/unified-agent/service

function install_init {
    cp -f $SERVICE_DIR/circonus-unified-agent.init /etc/init.d/circonus-unified-agent
    chmod +x /etc/init.d/circonus-unified-agent
}

function install_systemd {
    cp -f $SERVICE_DIR/circonus-unified-agent.service $1
    systemctl enable circonus-unified-agent || true
    systemctl daemon-reload || true
}

function install_update_rcd {
    update-rc.d circonus-unified-agent defaults
}

function install_chkconfig {
    chkconfig --add circonus-unified-agent
}

# Remove legacy symlink, if it exists
if [[ -L /etc/init.d/circonus-unified-agent ]]; then
    rm -f /etc/init.d/circonus-unified-agent
fi
# Remove legacy symlink, if it exists
if [[ -L /etc/systemd/system/circonus-unified-agent.service ]]; then
    rm -f /etc/systemd/system/circonus-unified-agent.service
fi

# Add defaults file, if it doesn't exist
if [[ ! -f /opt/circonus/unified-agent/etc/circonus-unified-agent.env ]]; then
    touch /opt/circonus/unified-agent/etc/circonus-unified-agent.env
fi

# Add .d configuration directory
if [[ ! -d /opt/circonus/unified-agent/etc/conf.d ]]; then
    mkdir -p /opt/circonus/unified-agent/etc/conf.d
fi

# If 'circonus-unified-agent.conf' is not present use package's sample (fresh install)
if [[ ! -f /opt/circonus/unified-agent/etc/circonus-unified-agent.conf ]] && [[ -f /opt/circonus/unified-agent/etc/example-circonus-unified-agent.conf ]]; then
   cp /opt/circonus/unified-agent/etc/example-circonus-unified-agent.conf /opt/circonus/unified-agent/etc/circonus-unified-agent.conf
fi

# Distribution-specific logic
if [[ -f /etc/redhat-release ]] || [[ -f /etc/SuSE-release ]]; then
    # RHEL-variant logic
    if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
        install_systemd /usr/lib/systemd/system/circonus-unified-agent.service
    else
        # Assuming SysVinit
        install_init
        # Run update-rc.d or fallback to chkconfig if not available
        if which update-rc.d &>/dev/null; then
            install_update_rcd
        else
            install_chkconfig
        fi
    fi
elif [[ -f /etc/os-release ]]; then
    source /etc/os-release
    if [[ "$NAME" = "Amazon Linux" ]]; then
        # Amazon Linux 2+ logic
        install_systemd /usr/lib/systemd/system/circonus-unified-agent.service
    elif [[ "$NAME" = "Amazon Linux AMI" ]]; then
        # Amazon Linux logic
        install_init
        # Run update-rc.d or fallback to chkconfig if not available
        if which update-rc.d &>/dev/null; then
            install_update_rcd
        else
            install_chkconfig
        fi
    elif [[ "$NAME" = "Solus" ]]; then
        # Solus logic
        install_systemd /usr/lib/systemd/system/circonus-unified-agent.service
    elif [[ "$ID" == *"sles"* ]] || [[ "$ID_LIKE" == *"suse"*  ]] || [[  "$ID_LIKE" = *"opensuse"* ]]; then
        # Modern SuSE logic
        install_systemd /usr/lib/systemd/system/circonus-unified-agent.service
    fi
fi
