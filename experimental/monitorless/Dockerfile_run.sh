#!/usr/bin/env bash

envvars=( HEALTH_PORT CRSTATES_PORT )
for v in $envvars; do
	if [[ -z $$v ]]; then echo "$v is unset"; exit 1; fi
done

ats_install_prefix=""
if [ -f "/opt/trafficserver/bin/trafficserver" ]; then
	ats_install_prefix="/opt/trafficserver"
fi

start() {
	nohup /astatstwo -debug -port="${HEALTH_PORT}" &
	nohup /healthcombiner -crconfig-path=/CRConfig.json -debug -port="${CRSTATES_PORT}" &
	# TODO run healthcombiner
	"${ats_install_prefix}/bin/trafficserver" start
	exec tail -f "${ats_install_prefix}/var/log/trafficserver/traffic.out"
}

init() {
	if [ ! -f /external/CRConfig.json ]; then
		echo "must run with a volume at /external/CRConfig.json"
		exit
	fi

	# copy the file, so we can change it without modify the host if we want
	cp /external/CRConfig.json /CRConfig.json
	ATS_PORT=`cat /CRConfig.json | jq ".contentServers.\"${HOSTNAME}\".port"`

	echo "CONFIG proxy.config.http.server_ports STRING ${ATS_PORT} ${ATS_PORT}:ipv6" >> "${ats_install_prefix}/etc/trafficserver/records.config"
	echo "CONFIG proxy.config.proxy_name STRING ${HOSTNAME}"  >> "${ats_install_prefix}/etc/trafficserver/records.config"

	# TODO run remapgen, place remaps and parents

	echo "INITIALIZED=1" >> /etc/environment
}

source /etc/environment
if [ -z "$INITIALIZED" ]; then init; fi
start
