#!/bin/bash

if ! grep "^cua:" /etc/group &>/dev/null; then
    groupadd -r cua
fi

if ! id cua &>/dev/null; then
    useradd -r -M cua -s /bin/false -d /opt/circonus/unified-agent -g cua
fi
