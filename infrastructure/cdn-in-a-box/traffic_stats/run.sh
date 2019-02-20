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

# Script for running the Dockerfile for Traffic Stats.
# The Dockerfile sets up a Docker image which can be used for any new container;
# This script, which should be run when the container is run (it's the ENTRYPOINT), will configure the container.
#
# The following environment variables must be set (ordinarily by `docker run -e` arguments):
# TO_HOST
# TO_PORT
# INFLUXDB_HOST

# Check that env vars are set

set -e
set -x
set -m

envvars=( TO_HOST TO_PORT INFLUXDB_HOST)
for v in $envvars
do
	if [[ -z $$v ]]; then echo "$v is unset"; exit 1; fi
done

source /to-access.sh

# Wait on SSL certificate generation
until [ -f "$CERT_DONE_FILE" ] 
do
  echo "Waiting on Shared SSL certificate generation"
  sleep 3
done

# Source the CIAB-CA shared SSL environment
source $CERT_ENV_FILE

# Trust the CIAB-CA at the System level
cp $CERT_CA_CERT_FILE /etc/pki/ca-trust/source/anchors
update-ca-trust extract

# Enroll with traffic ops
CDN=CDN-in-a-Box
TO_URL="https://$TO_FQDN:$TO_PORT"
to-enroll ts $CDN || (while true; do echo "enroll failed."; sleep 3 ; done)

while ! to-ping 2>/dev/null; do
	echo "waiting for trafficops ($TO_URL)..."
	sleep 3
done


export TO_USER=$TO_ADMIN_USER
export TO_PASSWORD=$TO_ADMIN_PASSWORD

# There's a race condition with setting the TM credentials and TO actually creating
# the TM user
until to-get "api/1.3/users?username=$TS_USER" 2>/dev/null | jq -c -e '.response[].username|length'; do
	echo "waiting for TS_USER creation..."
	sleep 3
done

# now that TS_USER is available,  use that for all further operations
export TO_USER="$TS_USER"
export TO_PASSWORD="$TS_PASSWORD"

export TO_USER=$TO_ADMIN_USER
export TO_PASSWORD=$TO_ADMIN_PASSWORD
envsubst </opt/traffic_stats/conf/traffic_stats.cfg.tmpl >/opt/traffic_stats/conf/traffic_stats.cfg

touch /opt/traffic_stats/var/log/traffic_stats.log

cd /opt/traffic_stats
/opt/traffic_stats/bin/traffic_stats -cfg /opt/traffic_stats/conf/traffic_stats.cfg || tail -f /dev/null
disown
exec tail -f /opt/traffic_stats/var/log/traffic_stats.log
