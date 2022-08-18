#!/usr/bin/env bash

set -e

#
# this script will update the external_plugins directory with fresh copies of each plugin
#

[[ -d external_plugins ]] || mkdir external_plugins
[[ -d tmp ]] || mkdir tmp

#
# Oracle - https://github.com/bonitoo-io/telegraf-input-oracle
#
[[ -d external_plugins/oracle ]] || mkdir external_plugins/oracle
pdir="$PWD/external_plugins/oracle"
pushd tmp > /dev/null
git clone https://github.com/bonitoo-io/telegraf-input-Oracle
pushd telegraf-input-Oracle > /dev/null
cp LICENSE $pdir/.
cp README.md $pdir/ORIG_README.md
cp oracle_metrics.py $pdir/.
cp oracle_metrics.sh $pdir/.
popd > /dev/null
rm -rf telegraf-input-Oracle
popd > /dev/null
