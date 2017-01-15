#!/bin/bash

docker_disconnect() {
    local network=$1
    local container=$2
    echo "Disconnecting ${container} from ${network}"
    docker network disconnect -f ${network} ${container}
}

docker_network_cleanup() {
    networks=$(docker network ls -q)
    for nw in ${networks}; do
        containers=$(docker network inspect --format='{{range .Containers}}{{.Name}} {{end}}' $nw)
        for ct in ${containers}; do
            docker inspect --format='{{.Id}}' ${ct} &> /dev/null || docker_disconnect ${nw} ${ct}
        done
    done
}

echo "Cleaning docker network..."
docker_network_cleanup
