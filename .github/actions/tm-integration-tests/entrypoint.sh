#!/bin/bash
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

set -e

function wait_for_endpoint() {
  try=0
  while [ $(curl -Lsk --write-out "%{http_code}" "$1" -o /dev/null) -ne 200 ] ; do
    echo "Waiting for $1 to return a 200 OK"
    try=$(expr $try + 1)
    if [[ $try -gt 5 ]]; then
      echo "Unable to get $1"
      exit 1
    fi
    sleep 5
  done
  echo "Able to reach $1"
}
export -f wait_for_endpoint

gray_bg="$(printf '%s%s' $'\x1B' '[100m')";
red_bg="$(printf '%s%s' $'\x1B' '[41m')";
yellow_bg="$(printf '%s%s' $'\x1B' '[43m')";
black_fg="$(printf '%s%s' $'\x1B' '[30m')";
color_and_prefix() {
	color="$1";
	shift;
	prefix="$1";
	normal_bg="$(printf '%s%s' $'\x1B' '[49m')";
	normal_fg="$(printf '%s%s' $'\x1B' '[39m')";
	sed "s/^/${color}${black_fg}${prefix}: /" | sed "s/$/${normal_bg}${normal_fg}/";
}

ciab_dir="${GITHUB_WORKSPACE}/infrastructure/cdn-in-a-box";

sudo apt-get install -y --no-install-recommends gettext

export GOPATH="${HOME}/go"
org_dir="$GOPATH/src/github.com/apache"
repo_dir="${org_dir}/trafficcontrol"
test_dir=${repo_dir}/traffic_monitor/tests/_integration
if [[ ! -e "$repo_dir" ]]; then
	mkdir -p "$org_dir"
	cd
	mv "${GITHUB_WORKSPACE}" "${repo_dir}/"
	ln -s "$repo_dir" "${GITHUB_WORKSPACE}"
fi

cd ${test_dir}
source variables.env
TM_PORT=8009
TESTTO_PORT=8000
PORT=8009
TO_URI=$(echo $TO_URI | sed 's/.*\/\/\(.*\)/\1/')
TM_URI=$(echo $TM_URI | sed 's/.*\/\/\(.*\)/\1/')

# Traffic Monitor runs /bin/hostname
sudo hostnamectl set-hostname $TM_URI
sudo hostname $TM_URI

sudo tee -a /etc/hosts  <<- EOF
127.0.0.1 $TO_URI
127.0.0.1 $TESTCACHES_URI
127.0.0.1 $TM_URI
EOF

jq ".[].tcpPort |= $PORT | .[].cdnName |= \"$CDN\" | .[].hostName |= \"$TM_URI\"" \
  ${test_dir}/servers.json > servers.json.tmp && mv servers.json.tmp ${test_dir}/servers.json

jq ".trafficMonitors[].port |= $PORT | .trafficMonitors[].ip6 |= \"$TM_URI\" | .trafficMonitors[].ip |= \"$TM_URI\" | .trafficMonitors[].hostName |= \"$TM_URI\" | .trafficServers[].interfaces[0].ipAddresses[0].address = \"127.0.0.1\"" \
  ${test_dir}/monitoring.json > monitoring.json.tmp && mv monitoring.json.tmp ${test_dir}/monitoring.json

jq ".monitors.trafficmonitor.port |= $PORT | .monitors.trafficmonitor.ip |= \"$TM_URI\" | .monitors.trafficmonitor.ip6 |= \"$TM_URI\" | .stats[\"CDN_name\"] = \"$CDN\" | .stats[\"tm_host\"] = \"$TESTTO_URI\"" \
  ${test_dir}/snapshot.json > snapshot.json.tmp && mv snapshot.json.tmp ${test_dir}/snapshot.json

cd "${repo_dir}/traffic_monitor/tools/testto"
go mod vendor
go build

touch to.log

./testto $TESTTO_PORT >to.log 2>&1 &
tail -f to.log | color_and_prefix "${gray_bg}" 'Test TO' &

wait_for_endpoint "${TESTTO_URI}:${TESTTO_PORT}/api/4.0/ping"

curl -Lvsk ${TESTTO_URI}:${TESTTO_PORT}/api/4.0/cdns/$CDN/snapshot -X POST -d "@${test_dir}/snapshot.json"
curl -Lvsk ${TESTTO_URI}:${TESTTO_PORT}/api/4.0/cdns/$CDN/configs/monitoring -X POST -d "@${test_dir}/monitoring.json"
curl -Lvsk ${TESTTO_URI}:${TESTTO_PORT}/api/4.0/servers -X POST -d "@${test_dir}/servers.json"

cd "${repo_dir}/traffic_monitor/tools/testcaches"
go mod vendor
go build

touch cache.log

./testcaches -numPorts "$NUM_PORTS" -numRemaps "$NUM_REMAPS" -portStart $TESTCACHES_PORT_START > cache.log 2>&1 &
tail -f cache.log | color_and_prefix "${gray_bg}" 'Test Cache' &

wait_for_endpoint "$TESTCACHES_URI:$TESTCACHES_PORT_START/_astats"

cd "${repo_dir}/traffic_monitor"
go mod vendor
go build

cat > ./traffic_monitor.cfg <<- EOF
  {
      "cache_health_polling_interval_ms": 6000,
      "cache_stat_polling_interval_ms": 6000,
      "monitor_config_polling_interval_ms": 15000,
      "http_timeout_ms": 8000,
      "peer_polling_interval_ms": 5000,
      "peer_optimistic": true,
      "max_events": 200,
      "max_stat_history": 5,
      "max_health_history": 5,
      "health_flush_interval_ms": 20,
      "stat_flush_interval_ms": 20,
      "log_location_event": null,
      "log_location_error": "err.log",
      "log_location_warning": "warn.log",
      "log_location_info": null,
      "log_location_debug": null,
      "serve_read_timeout_ms": 10000,
      "serve_write_timeout_ms": 10000,
      "http_poll_no_sleep": false,
      "static_file_dir": "${repo_dir}/traffic_monitor/static/"
  }
EOF
touch err.log warn.log
tail -f warn.log 2>&1 | color_and_prefix "${yellow_bg}" "Traffic Monitor" &
tail -f err.log 2>&1 | color_and_prefix "${red_bg}" "Traffic Monitor" &

cat > ./traffic_ops.cfg <<- EOF
  {
      "username": "$TO_USER",
      "password": "$TO_PASS",
      "url": "http://$TO_URI:$TESTTO_PORT",
      "insecure": true,
      "cdnName": "$CDN",
      "httpListener": ":$PORT"
      }
EOF

./traffic_monitor -opsCfg traffic_ops.cfg -config traffic_monitor.cfg  &
sleep 25s


wait_for_endpoint "http://$TM_URI:$TM_PORT/api/version"

cd ${test_dir}

jq ".trafficMonitor.URL = \"http://$TM_URI:$TM_PORT\"" \
  traffic-monitor-test.conf > tmt.conf.tmp && mv tmt.conf.tmp traffic-monitor-test.conf

go test -cfg traffic-monitor-test.conf
