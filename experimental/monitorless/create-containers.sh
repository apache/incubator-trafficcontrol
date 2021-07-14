#!/usr/bin/env bash

# set -e          # exit if any command has a nonzero exit code
# set -u          # exit if any undefined variable is referenced
# set -o pipefail # exit if any command in a pipeline is nonzero # commented, because sometimes you want to || true

#
# Creates and configures docker containers for monitorless monitoring.
# Requires:
# - jq
# - the working directory be experimental/monitorless
# - A docker image named 'mm' exist. Run 'docker build --no-cache --rm --tag mm:0.1 .'
# - A CRConfig.json at ./CRConfig.json
#    - Everything in the CRConfig.json needs to be correct, except the IPs
#      This script will create containers for all servers in .contentServers
#      It will copy the CRConfig to crconfig-mm.json, and then modify it to have the docker container IPs
# - A docker network named 'mm'.
#

if [[ "$1" == "clean" ]]; then
	echo "removing containers: "
	container_names=$(cat CRConfig.json | jq -r '.contentServers | keys | .[]')
	container_names_spaced=$(echo $container_names)
	docker rm -f $container_names_spaced
	rm ./mm-crconfig.json
	exit 0
fi

# Note this script copies ./CRConfig.json into ./mm-crconfig.json and modifies it for the created containers

if [ -z "${BASH_VERSINFO}" ] || [ -z "${BASH_VERSINFO[0]}" ] || [ ${BASH_VERSINFO[0]} -lt 4 ]; then
	printf "This script requires Bash version >= 4\n";
	exit 1;
fi

# TODO create docker image if it doesn't exist
# TODO create docker network if it doesn't exist

printf "running bash 4+\n"

# configurable variables:
health_port=8089
crstates_port=8088

container_names=$(cat CRConfig.json | jq -r '.contentServers | keys | .[]')

cp ./CRConfig.json ./mm-crconfig.json

# this is slightly insane, but because docker exec reads from stdin, the better read line loop doesn't work
for container_name in ${container_names//\\n/ }; do
	printf "creating container '${container_name}'\n"
	# TODO exclude non-reported/online? non-edges?

	docker run --detach --name "${container_name}" --hostname "${container_name}" --net mm --env HEALTH_PORT="${health_port}" --env CRSTATES_PORT="${crstates_port}" --volume $(pwd)/mm-crconfig.json:/external/CRConfig.json -- mm:0.1

	# Set this server's IP in the CRConfig to the docker container's IP
	ip=$(docker exec -i "${container_name}" sh -c 'ip addr' | grep -A 2 ': eth0' | tail -1 | awk '{print $2}' | cut -f1 -d'/')
	cat mm-crconfig.json | jq "(.contentServers.\"${container_name}\".ip ) = \"${ip}\"" > mm-crconfig.json.tmp
	mv mm-crconfig.json{.tmp,}
	printf "ip: ${ip}\n"

done

# after all containers are created, and their IPs put in the CRConfig, run remapgen in the containers

for container_name in ${container_names//\\n/ }; do
	ats_install_prefix=""
	docker exec -i "${container_name}" sh -c "test -f /opt/trafficserver/bin/trafficserver"
	ats_in_opt=$?
	printf "container ${container_name} in opt: ${ats_in_opt}\n"
	if [ "$ats_in_opt" = "0" ]; then
		ats_install_prefix="/opt/trafficserver"
	fi
	printf "container ${container_name} prefix: '${ats_install_prefix}'\n"

	# re-copy the external CRConfig that we just added IPs to, to prevent accidental modification
	docker exec -i "${container_name}" sh -c "cat /external/CRConfig.json > /CRConfig.json"

	docker exec -i "${container_name}" sh -c "/remapgen -host ${container_name} -crconfig-path /CRConfig.json -health-port \${HEALTH_PORT} -comments=true -config remap > ${ats_install_prefix}/etc/trafficserver/remap.config"

	docker exec -i "${container_name}" sh -c "/remapgen -host ${container_name} -crconfig-path /CRConfig.json -health-port \${HEALTH_PORT} -comments=true -config parent > ${ats_install_prefix}/etc/trafficserver/parent.config"

	docker exec -i "${container_name}" sh -c "pkill traffic_manager && pkill '[TS_MAIN]' && sleep 1 && ${ats_install_prefix}/bin/trafficserver start"
#	docker exec -i "${container_name}" sh -c '${ats_install_prefix}/trafficserver/bin/trafficserver restart'
done
