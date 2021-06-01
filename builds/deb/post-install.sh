#!/bin/bash

BIN_DIR=/opt/circonus/unified-agent/sbin
SERVICE_DIR=/opt/circonus/unified-agent/service

function install_init {
    cp -f $SERVICE_DIR/init.sh /etc/init.d/circonus-unified-agent
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
if [[ ! -d /opt/circonus/unified-agent/etc/config.d ]]; then
    mkdir -p /opt/circonus/unified-agent/etc/config.d
fi

# If 'circonus-unified-agent.conf' is not present use package's sample (fresh install)
if [[ ! -f /opt/circonus/unified-agent/etc/circonus-unified-agent.conf ]] && [[ -f /opt/circonus/unified-agent/etc/example-circonus-unified-agent.conf ]]; then
   cp /opt/circonus/unified-agent/etc/example-circonus-unified-agent.conf /opt/circonus/unified-agent/etc/circonus-unified-agent.conf
fi

if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
	install_systemd /lib/systemd/system/circonus-unified-agent.service
	deb-systemd-invoke restart circonus-unified-agent.service || echo "WARNING: systemd not running."
else
	# Assuming SysVinit
	install_init
	# Run update-rc.d or fallback to chkconfig if not available
	if which update-rc.d &>/dev/null; then
		install_update_rcd
	else
		install_chkconfig
	fi
	invoke-rc.d circonus-unified-agent restart
fi
