#!/usr/bin/env bash
#
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

# Required env vars
# Check that env vars are set
set -ex
for v in TO_HOST TO_PORT TO_ADMIN_USER TO_ADMIN_PASSWORD; do
    [[ -z $(eval echo \$$v) ]] || continue
    echo "$v is unset"
    exit 1
done

. /to-access.sh

TO_URL="https://$TO_FQDN:$TO_PORT"
# wait until the ping endpoint succeeds
while ! to-ping 2>/dev/null; do
   echo waiting for trafficops
   sleep 3
done

# NOTE: order dependent on foreign key references, e.g. profiles must be loaded before parameters
endpoints="cdns types divisions regions phys_locations tenants users cachegroups profiles parameters server_capabilities servers topologies deliveryservices server_server_capabilities deliveryservice_servers deliveryservices_required_capabilities"
vars=$(awk -F = '/^\w/ {printf "$%s ",$1}' /variables.env)

waitfor() {
    local endpoint="$1"; shift
    local field="$1"; shift
    local value="$1"; shift
    local responseField="$1"
    if [[ -z "$responseField" ]]; then
      responseField="$field"
    else
      shift
    fi
    local additionalQueryString="$1"
    if [[ -n "$additionalQueryString" ]]; then
      shift
    fi

    while true; do
        v="$(to-get "api/${TO_API_VERSION}/${endpoint}?${field}=${value}${additionalQueryString}" | jq -r --arg field "$responseField" '.response[][$field]')";
        if [[ "$v" == "$value" ]]; then
          break
        fi
        echo "waiting for $endpoint $field=$value"
        sleep 3
    done
}

# special cases -- any data type requiring specific data to already be available in TO should have an entry here.
# e,g. deliveryservice_servers requires both deliveryservice and all servers to be available
delayfor() {
    local f="$1"
    local d="${f%/*}"

    case $d in
        deliveryservices)
            local cdns_json
            cdns_json="$(to-get "api/${TO_API_VERSION}/cdns?name=${CDN_NAME}")"
            if [[ "$(<<<"$cdns_json" jq '.response[].dnssecEnabled')" == false ]]; then
              echo "DNSSEC is not enabled for ${CDN_FQDN}"
              return
            fi
            jq_filter='.date = $date | .key = $cdn_name | .name = $domain | .ttl = 60 | .kskExpirationDays = 365 | .zskExpirationDays = 30'
            generate_dnssec_body="$(jq -n --arg cdn_name "$CDN_NAME" --arg domain "$CDN_FQDN" --arg date "$(date --rfc-3339=seconds | sed 's/ /T/')" "$jq_filter")"
            until to-post "api/${TO_API_VERSION}/cdns/dnsseckeys/generate" "$generate_dnssec_body"; do
              echo "Waiting for DNSSEC key generation request to succeed for ${CDN_FQDN}..."
              sleep 3
            done
              echo "Generated DNSSEC keys for ${CDN_FQDN}"
            ;;
        deliveryservice_servers)
            ds=$( jq -r .xmlId <"$f" )
            waitfor deliveryservices xmlId "$ds"
            for s in $( jq -r .serverNames[] <"$f" ); do
                waitfor servers hostName "$s"
            done
            ;;
        topologies)
            for cachegroup_name in $(jq -r '.nodes[] | .cachegroup' <"$f"); do
              waitfor cachegroups name "$cachegroup_name"
              cachegroup="$(to-get "api/${TO_API_VERSION}/cachegroups?name=${cachegroup_name}")"
              cachegroup_id="$(<<<"$cachegroup" jq '.response[] | .id')"
              cachegroup_type="$(<<<"$cachegroup" jq -r '.response[] | .typeName')"
              waitfor servers cachegroup "$cachegroup_id" cachegroupId "&type=${cachegroup_type%_LOC}"
            done
            ;;
    esac
}

load_data_from() {
    local dir="$1"
    if [[ ! -d $dir ]] ; then
        echo "Failed to load data from '$dir': directory does not exist"
    fi
    cd "$dir"

    local status=0
    local has_ds_servers=''
    if ls deliveryservice_servers/'*.json'; then
      has_ds_servers='true'
    fi
    for d in $endpoints; do
        # Let containers know to write out server.json
        if [[ "$d" = 'topologies' ]]; then
           touch "$ENROLLER_DIR/initial-load-done"
           sync
        fi
        [[ -d $d ]] || continue
        for f in $(find "$d" -name "*.json" -type f); do
            echo "Loading $f"
            if [ ! -f "$f" ]; then
              echo "No such file: $f" >&2;
              continue;
            fi
            delayfor "$f"
            envsubst "$vars" <"$f"  > "$ENROLLER_DIR/$f"
            sync
        done
    done
    if [[ $status -ne 0 ]]; then
        exit $status
    fi
    cd -
}

# If Traffic Router debugging is enabled, keep zone generation from timing out
# (for 5 minutes), in case that is what you are debugging
traffic_router_zonemanager_timeout() {
  if [[ "$TR_DEBUG_ENABLE" != true ]]; then
    return;
  fi;

  local modified_crconfig crconfig_path zonemanager_timeout;
  crconfig_path=/traffic_ops_data/profiles/040-CCR_CIAB.json;
  modified_crconfig="$(mktemp)";
  # 5 minutes, which is the default zonemanager.cache.maintenance.interval value
  zonemanager_timeout="$(( 60 * 5 ))";
  jq \
    --arg zonemanager_timeout $zonemanager_timeout \
    '.params = .params + [{"configFile": "CRConfig.json", "name": "zonemanager.init.timeout", "value": $zonemanager_timeout}]' \
    <$crconfig_path >"$modified_crconfig";
  mv "$modified_crconfig" $crconfig_path;
}

if [[ "$LOAD_TRAFFIC_OPS_DATA" == true ]]; then
	traffic_router_zonemanager_timeout

	# Load required data at the top level
	load_data_from /traffic_ops_data
else
	touch "$ENROLLER_DIR/initial-load-done"
fi
