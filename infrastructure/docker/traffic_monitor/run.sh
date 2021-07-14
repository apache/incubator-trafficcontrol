#!/usr/bin/env bash
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

# Script for running the Dockerfile for Traffic Monitor.
# The Dockerfile sets up a Docker image which can be used for any new container;
# This script, which should be run when the container is run (it's the ENTRYPOINT), will configure the container.
#
# The following environment variables must be set (ordinarily by `docker run -e` arguments):
# TRAFFIC_OPS_URI
# TRAFFIC_OPS_USER
# TRAFFIC_OPS_PASS

# Check that env vars are set
envvars=( TRAFFIC_OPS_URI TRAFFIC_OPS_USER TRAFFIC_OPS_PASS )
for v in $envvars
do
	if [[ -z $$v ]]; then echo "$v is unset"; exit 1; fi
done

if [ -z "$NO_WAIT" ]; then
		TO_COOKIE="$(curl -v -s -k -X POST --data '{ "u":"'"$TRAFFIC_OPS_USER"'", "p":"'"$TRAFFIC_OPS_PASS"'" }' $TRAFFIC_OPS_URI/api/1.2/user/login 2>&1 | grep 'Set-Cookie' | sed -e 's/.*mojolicious=\(.*\); expires.*/\1/')"
		while [ -z "$TO_COOKIE" ]; do
				echo "Waiting for Traffic Ops to start..."
				sleep 1;
				TO_COOKIE="$(curl -v -s -k -X POST --data '{ "u":"'"$TRAFFIC_OPS_USER"'", "p":"'"$TRAFFIC_OPS_PASS"'" }' $TRAFFIC_OPS_URI/api/1.2/user/login 2>&1 | grep 'Set-Cookie' | sed -e 's/.*mojolicious=\(.*\); expires.*/\1/')"
		done
		echo "Traffic Ops Started (cookie $TO_COOKIE) - Starting Monitor Now!"
fi

start() {
	service traffic_monitor start
	touch /opt/traffic_monitor/var/log/traffic_monitor.log
	exec tail -f /opt/traffic_monitor/var/log/traffic_monitor.log
}

init() {
	[ ! -z $IP ]            || IP=$(ip addr | grep 'global' | awk '{print $2}' | cut -f1 -d'/')
	[ ! -z $DOMAIN ]        || DOMAIN="localdomain"
	[ ! -z $CACHEGROUP ]    || CACHEGROUP="mid-east"
	[ ! -z $TYPE ]          || TYPE="RASCAL"
	[ ! -z $PROFILE ]       || PROFILE="RASCAL_CDN1"
	[ ! -z $PHYS_LOCATION ] || PHYS_LOCATION="plocation-nyc-1"
	[ ! -z $INTERFACE ]     || INTERFACE="eth0"
	[ ! -z $NETMASK ]       || NETMASK="255.255.0.0"
	[ ! -z $MTU ]           || MTU="9000"
	[ ! -z $PORT ]          || PORT="80"
	[ ! -z $GATEWAY ]       || GATEWAY="$(ip route | grep default | awk '{print $3}')"
	[ ! -z $CDN ]           || CDN="cdn"
	echo "IP: $IP"
	echo "Domain: $DOMAIN"
	echo "Cachegroup: $CACHEGROUP"
	echo "Type: $TYPE"
	echo "Profile: $PROFILE"
	echo "PhysLocation: $PHYS_LOCATION"
	echo "Interface: $INTERFACE"
	echo "NetMask: $NETMASK"
	echo "MTU: $MTU"
	echo "Port: $PORT"
	echo "Gateway: $GATEWAY"
	echo "CDN: $CDN"
	echo "Create Server: $CREATE_TO_SERVER"

	mkdir -p /opt/traffic_monitor/conf
	cat > /opt/traffic_monitor/conf/traffic_monitor.cfg <<- ENDOFMESSAGE
		{
				"cache_health_polling_interval_ms": 6000,
				"cache_stat_polling_interval_ms": 6000,
				"monitor_config_polling_interval_ms": 15000,
				"http_timeout_ms": 2000,
				"peer_polling_interval_ms": 5000,
				"peer_optimistic": true,
				"max_events": 200,
				"max_stat_history": 5,
				"max_health_history": 5,
				"health_flush_interval_ms": 20,
				"stat_flush_interval_ms": 20,
				"log_location_event": "/opt/traffic_monitor/var/log/event.log",
				"log_location_error": "/opt/traffic_monitor/var/log/traffic_monitor.log",
				"log_location_warning": "/opt/traffic_monitor/var/log/traffic_monitor.log",
				"log_location_info": "null",
				"log_location_debug": "null",
				"serve_read_timeout_ms": 10000,
				"serve_write_timeout_ms": 10000,
				"http_poll_no_sleep": false,
				"static_file_dir": "/opt/traffic_monitor/static/",
				"peer_polling_protocol": "both",
				"cache_polling_protocol": "both"
		}
ENDOFMESSAGE

	cat > /opt/traffic_monitor/conf/traffic_ops.cfg <<- ENDOFMESSAGE
		{
				"username": "$TRAFFIC_OPS_USER",
				"password": "$TRAFFIC_OPS_PASS",
				"url": "$TRAFFIC_OPS_URI",
				"insecure": true,
				"cdnName": "$CDN",
				"httpListener": ":$PORT"
				}
	ENDOFMESSAGE

	TO_COOKIE="$(curl -v -s -k -X POST --data '{ "u":"'"$TRAFFIC_OPS_USER"'", "p":"'"$TRAFFIC_OPS_PASS"'" }' $TRAFFIC_OPS_URI/api/1.2/user/login 2>&1 | grep 'Set-Cookie' | sed -e 's/.*mojolicious=\(.*\); expires.*/\1/')"
	echo "Got Cookie: $TO_COOKIE"

	if [ ! -z "$CREATE_TO_SERVER" ] ; then
		echo "Creating Server in Traffic Ops!"
		# curl -v -k -X POST -H "Cookie: mojolicious=$TO_COOKIE" -F "filename=Traffic_Monitor_Dockerfile_profile.traffic_ops" -F "profile_to_import=@/Traffic_Monitor_Dockerfile_profile.traffic_ops" $TRAFFIC_OPS_URI/profile/doImport

		CACHEGROUP_ID="$( curl -s -k -X GET -H "Cookie: mojolicious=$TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/cachegroups.json | jq '.response | .[] | select(.name=='"\"$CACHEGROUP\""') | .id')"
		echo "Got cachegroup ID: $CACHEGROUP_ID"

		SERVER_TYPE_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/types.json | jq '.response | .[] | select(.name=='"\"$TYPE\""') | .id')"
		echo "Got server type ID: $SERVER_TYPE_ID"

		SERVER_PROFILE_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/profiles.json | jq '.response | .[] | select(.name=='"\"$PROFILE\""') | .id')"
		echo "Got server profile ID: $SERVER_PROFILE_ID"

		PHYS_LOCATION_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/phys_locations.json | jq '.response | .[] | select(.shortName=='"\"$PHYS_LOCATION\""') | .id')"
		echo "Got phys location ID: $PHYS_LOCATION_ID"

		CDN_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/cdns.json | jq '.response | .[] | select(.name=='"\"$CDN\""') | .id')"
		echo "Got cdn ID: $CDN_ID"

		# Create Server in Traffic Ops
		curl -v -k -X POST -H "Cookie: mojolicious=$TO_COOKIE" --data-urlencode "host_name=$HOSTNAME" --data-urlencode "domain_name=$DOMAIN" --data-urlencode "interface_name=$INTERFACE" --data-urlencode "ip_address=$IP" --data-urlencode "ip_netmask=$NETMASK" --data-urlencode "ip_gateway=$GATEWAY" --data-urlencode "interface_mtu=$MTU" --data-urlencode "cdn=$CDN_ID" --data-urlencode "cachegroup=$CACHEGROUP_ID" --data-urlencode "phys_location=$PHYS_LOCATION_ID" --data-urlencode "type=$SERVER_TYPE_ID" --data-urlencode "profile=$SERVER_PROFILE_ID" --data-urlencode "tcp_port=$PORT" --data-urlencode "offline_reason=creation" $TRAFFIC_OPS_URI/server/create

		# Add Monitor IP to `allow_ip` Parameters
		IP_ALLOW_PARAMS=$(curl -Lsk --cookie "mojolicious=$TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/parameters?name=allow_ip | jq '.response | .[] | .id, .value')
		while IFS= read -r id; do
			IFS= read -r ipallow
			ipallow=$(echo ${ipallow} | sed -e 's/^"//' -e 's/"$//')
			IPALLOW_UPDATE_JSON="{\"id\": ${id}, \"value\": \"${ipallow},${IP}\"}"
			curl -Lsk --cookie "mojolicious=$TO_COOKIE" -H 'Content-Type: application/json' -X PUT -d "$IPALLOW_UPDATE_JSON" $TRAFFIC_OPS_URI/api/1.2/parameters/${id}
		done <<< "$IP_ALLOW_PARAMS"
	fi

	SERVER_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/servers.json | jq '.response | .[] | select(.hostName=='"\"$HOSTNAME\""') | .id')"
	echo "Got server ID: $SERVER_ID"

	# Set Server to Online in Traffic Ops
	curl -v -k -H "Content-Type: application/x-www-form-urlencoded" -H "Cookie: mojolicious=$TO_COOKIE" -X POST --data-urlencode "id=$SERVER_ID" --data-urlencode "status=ONLINE" $TRAFFIC_OPS_URI/server/updatestatus

	echo "INITIALIZED=1" >> /etc/environment
}

source /etc/environment
if [ -z "$INITIALIZED" ]; then init; fi
start
