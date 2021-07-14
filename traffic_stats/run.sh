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

echo "Starting Traffic Stats"
set -e
set -x
set -m

envvars=( TO_HOST TO_PORT INFLUXDB_HOST )
for v in $envvars
do
  if [[ -z "${!v}" ]]; then echo "$v is unset"; exit 1; fi
done

# Enroll with traffic ops
TSCONF=/opt/traffic_stats/conf/traffic_stats.cfg

#while ! to-ping 2>/dev/null; do
#  echo "waiting for trafficops ($TO_URL)..."
#  sleep 3
#done

cat <<-EOF >$TSCONF
{
	"toUser": "$TO_ADMIN_USER",
	"toPasswd": "$TO_ADMIN_PASSWORD",
	"toUrl": "$TO_URL",
	"influxUser": "$INFLUXDB_ADMIN_USER",
	"influxPassword": "$INFLUXDB_ADMIN_PASSWORD",
	"pollingInterval": 10,
	"publishingInterval": 30,
	"maxPublishSize": 10000,
	"statusToMon": "ONLINE",
	"seelogConfig": "/opt/traffic_stats/conf/traffic_stats_seelog.xml",
	"dailySummaryPollingInterval": 300,
	"cacheRetentionPolicy": "daily",
	"dsRetentionPolicy": "daily",
	"dailySummaryRetentionPolicy": "indefinite",
    "influxUrls": ["http://$INFLUXDB_HOST:$INFLUXDB_PORT"]
}
EOF

cat <<-EOF >/opt/traffic_stats/conf/traffic_stats_seelog.xml
<?xml version='1.0'?>
<seelog minlevel="debug">
    <outputs formatid="std:debug-short">
        <console/>
    </outputs>
</seelog>
EOF

touch /opt/traffic_stats/var/log/traffic_stats.log

traffic_stats_command=(/opt/traffic_stats/bin/traffic_stats -cfg $TSCONF);
if [[ "$TS_DEBUG_ENABLE" == true ]]; then
  dlv '--continue' '--listen=:2346' '--accept-multiclient=true' '--headless=true' '--api-version=2' exec \
    "${traffic_stats_command[0]}" -- "${traffic_stats_command[@]:1}" &
else
  "${traffic_stats_command[@]}"
fi;

#exec tail -f /opt/traffic_stats/var/log/traffic_stats.log
