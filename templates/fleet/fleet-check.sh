#!/bin/bash

DRYRUN=""

check_inactive_units() {
    inactiveUnits=$(fleetctl list-units -no-legend | grep "\.service" | grep -v running | awk '{print $1}' | uniq)

    restartUnits=""
    for unit in ${inactiveUnits}; do
        oneshot=$(fleetctl cat ${unit} | grep "Type=oneshot")
        if [ -z "$oneshot" ]; then
            restartUnits="$restartUnits $unit"
            echo "Found inactive ${unit}"
        fi
    done

    if [ -z "$DRYRUN" ]; then
        for unit in ${restartUnits}; do
            fleetctl stop ${unit}
        done
        for unit in ${restartUnits}; do
            fleetctl start ${unit}
        done
    fi
}

for i in "$@"
do
case $i in
    --dry-run)
    DRYRUN=1
    ;;
    *)
    echo "unknown option '${i}'"
    exit 1
    ;;
esac
done

check_inactive_units
