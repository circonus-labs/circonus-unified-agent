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

if [ "$1" == "remove" -o "$1" == "purge" ]; then
	if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
		disable_systemd /lib/systemd/system/circonus-unified-agent.service
	else
		# Assuming sysv
		# Run update-rc.d or fallback to chkconfig if not available
		if which update-rc.d &>/dev/null; then
			disable_update_rcd
		else
			disable_chkconfig
		fi
	fi
fi
