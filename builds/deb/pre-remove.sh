#!/bin/bash

if [[ "$(readlink /proc/1/exe)" == */systemd ]]; then
	deb-systemd-invoke stop circonus-unified-agent.service
else
	# Assuming sysv
	invoke-rc.d circonus-unified-agent stop
fi
