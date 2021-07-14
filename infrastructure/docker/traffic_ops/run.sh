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

# Script for running the Dockerfile for Traffic Ops.
# The Dockerfile sets up a Docker image which can be used for any new Traffic Ops container;
# This script, which should be run when the container is run (it's the ENTRYPOINT), will configure the container.
#
# The following environment variables must be set, ordinarily by `docker run -e` arguments:
# DB_SERVER
# DB_PORT
# DB_ROOT_PASS
# DB_USER
# DB_USER_PASS
# DB_NAME
# ADMIN_USER
# ADMIN_PASS
# CERT_COUNTRY
# CERT_STATE
# CERT_CITY
# CERT_COMPANY
# DOMAIN

# Executes the given argument as a command, every second, until they succeed, up to retrys seconds. If retrys elapses, and there is no success, this exits the script with a nonzero exit code.
# TODO remove duplication, put in utils script for Dockerfiles
function retry() {
	local retrys=300

	local code=0
	while true; do
		"$@"
		code=$?
		# echo "curl 0returned $?"
		if [[ $code -eq 0 ]]; then
			break
		fi
		if [[ retrys -eq 0 ]]; then
			break
		fi
		sleep 1;
		retrys=$[$retrys-1]
	done
	if [[ $code -ne 0 ]]; then
		exit 1
	fi
}

# Returns whether the Postgres database is ready. Tries, sleeps, and tries again, because the Postgres Dockerfile restarts several times.
function dbready() {
	echo "Waiting for database..."
	# EXISTS=$(PGPASSWORD=${DB_USER_PASS} psql -h ${DB_SERVER} -U ${DB_USER} -d traffic_ops -c "COPY (select count(*) from tm_user where username = '${ADMIN_USER}') TO STDOUT")
	EXISTS=$(PGPASSWORD=${DB_USER_PASS} psql -h ${DB_SERVER} -U ${DB_USER} -c "COPY (select 1 from pg_database where datname='postgres') TO STDOUT")
	echo "Database exists: ${EXISTS}"
	if [[ ${EXISTS} -ne 1 ]]; then
		return 1
	fi
	sleep 2
	EXISTS=$(PGPASSWORD=${DB_USER_PASS} psql -h ${DB_SERVER} -U ${DB_USER} -c "COPY (select 1 from pg_database where datname='postgres') TO STDOUT")
	echo "Database exists: ${EXISTS}"
	if [[ ${EXISTS} -ne 1 ]]; then
		return 1
	fi
	return 0
}

# Check that env vars are set
envvars=( DB_SERVER DB_PORT DB_ROOT_PASS DB_USER DB_USER_PASS ADMIN_USER ADMIN_PASS CERT_COUNTRY CERT_STATE CERT_CITY CERT_COMPANY DOMAIN)
for v in $envvars
do
	if [[ -z $$v ]]; then echo "$v is unset"; exit 1; fi
done

if [ -z "$NO_WAIT" ]; then
	retry dbready
	code=$?
	if [[ $code -ne 0 ]]; then
		echo "Failed to get database connection, cannot start!"
		exit 1
	fi
fi

start() {
	service traffic_ops start
	exec tail -f /var/log/traffic_ops/traffic_ops.log
}

init() {
	local postinstall_input_file="postinstall-input.json"
	cat > "$postinstall_input_file" <<- ENDOFMESSAGE
{
  "/opt/traffic_ops/app/conf/production/database.conf":[
    {
      "Database type":"Pg",
      "config_var":"type"
    },
    {
      "Database name":"$DB_NAME",
      "config_var":"dbname"
    },
    {
      "Database server hostname IP or FQDN":"$DB_SERVER",
      "config_var":"hostname"
    },
    {
      "Database port number":"$DB_PORT",
      "config_var":"port"
    },
    {
      "Traffic Ops database user":"$DB_USER",
      "config_var":"user"
    },
    {
      "Traffic Ops database password":"$DB_USER_PASS",
      "config_var":"password",
      "hidden":"1"
    }
  ],
  "/opt/traffic_ops/app/db/dbconf.yml":[
    {
      "Database server root (admin) user":"postgres",
      "config_var":"pgUser"
    },
    {
      "Database server admin password":"$DB_ROOT_PASS",
      "config_var":"pgPassword",
      "hidden":"1"
    },
    {
      "Download Maxmind Database?":"yes",
      "config_var":"maxmind"
    }
  ],
  "/opt/traffic_ops/app/conf/cdn.conf":[
    {
      "Generate a new secret?":"yes",
      "config_var":"genSecret"
    },
    {
      "Port to serve on?": "443",
      "config_var": "port"
    },
    {
      "Number of workers?": "12",
      "config_var":"workers"
    },
    {
      "Traffic Ops url?": "https://$HOSTNAME",
      "config_var": "base_url"
    },
    {
      "Number of secrets to keep?":"1",
      "config_var":"keepSecrets"
    }
  ],
  "/opt/traffic_ops/app/conf/ldap.conf":[
    {
      "Do you want to set up LDAP?":"no",
      "config_var":"setupLdap"
    },
    {
      "LDAP server hostname":"",
      "config_var":"host"
    },
    {
      "LDAP Admin DN":"",
      "config_var":"admin_dn"
    },
    {
      "LDAP Admin Password":"",
      "config_var":"admin_pass",
      "hidden":"1"
    },
    {
      "LDAP Search Base":"",
      "config_var":"search_base"
    }
  ],
  "/opt/traffic_ops/install/data/json/users.json":[
    {
      "Administration username for Traffic Ops":"$ADMIN_USER",
      "config_var":"tmAdminUser"
    },
    {
      "Password for the admin user":"$ADMIN_PASS",
      "config_var":"tmAdminPw",
      "hidden":"1"
    }
  ],
  "/opt/traffic_ops/install/data/profiles/":[
    {
      "Add custom profiles?":"no",
      "config_var":"custom_profiles"
    }
  ],
  "/opt/traffic_ops/install/data/json/openssl_configuration.json":[
    {
      "Do you want to generate a certificate?":"yes",
      "config_var":"genCert"
    },
    {
      "Country Name (2 letter code)":"$CERT_COUNTRY",
      "config_var":"country"
    },
    {
      "State or Province Name (full name)":"$CERT_STATE",
      "config_var":"state"
    },
    {
      "Locality Name (eg, city)":"$CERT_CITY",
      "config_var":"locality"
    },
    {
      "Organization Name (eg, company)":"$CERT_COMPANY",
      "config_var":"company"
    },
    {
      "Organizational Unit Name (eg, section)":"",
      "config_var":"org_unit"
    },
    {
      "Common Name (eg, your name or your server's hostname)":"$HOSTNAME",
      "config_var":"common_name"
    },
    {
      "RSA Passphrase":"passphrase",
      "config_var":"rsaPassword",
      "hidden":"1"
    }
  ],
  "/opt/traffic_ops/install/data/json/profiles.json":[
    {
      "Traffic Ops url":"https://$HOSTNAME",
      "config_var":"tm.url"
    },
    {
      "Human-readable CDN Name.  (No whitespace, please)":"cdn",
      "config_var":"cdn_name"
    },
    {
      "Health Polling Interval (milliseconds)":"8000",
      "config_var":"health_polling_int"
    },
    {
      "DNS sub-domain for which your CDN is authoritative":"$HOSTNAME.$DOMAIN",
      "config_var":"dns_subdomain"
    },
    {
      "TLD SOA admin":"traffic_ops",
      "config_var":"soa_admin"
    },
    {
      "TrafficServer Drive Prefix":"/dev/ram",
      "config_var":"driver_prefix"
    },
    {
      "TrafficServer RAM Drive Prefix":"/dev/ram",
      "config_var":"ram_drive_prefix"
    },
    {
      "TrafficServer RAM Drive Letters (comma separated)":"1",
      "config_var":"ram_drive_letters"
    },
    {
      "Health Threshold Load Average":"25",
      "config_var":"health_thresh_load_avg"
    },
    {
      "Health Threshold Available Bandwidth in Kbps":"1750000",
      "config_var":"health_thresh_kbps"
    },
    {
      "Traffic Server Health Connection Timeout (milliseconds)":"2000",
      "config_var":"health_connect_timeout"
    }
  ]
}
	ENDOFMESSAGE

	# TODO determine if term, user are necessary
	export TERM=xterm && export USER=root && /opt/traffic_ops/install/bin/postinstall -cfile "$postinstall_input_file"

	# Only listen on IPv4, not IPv6, because Docker doesn't provide a v6 interface by default. See http://mojolicious.org/perldoc/Mojo/Server/Daemon#listen
	sed -i -e 's#https://\[::\]#https://127\.0\.0\.1#' /opt/traffic_ops/app/conf/cdn.conf
	service traffic_ops restart

	TRAFFIC_OPS_URI="https://localhost"

	TMP_TO_COOKIE="$(curl -v -s -k -X POST --data '{ "u":"'"$ADMIN_USER"'", "p":"'"$ADMIN_PASS"'" }' $TRAFFIC_OPS_URI/api/1.2/user/login 2>&1 | grep 'Set-Cookie' | sed -e 's/.*mojolicious=\(.*\); expires.*/\1/')"
	echo "Got cookie: $TMP_TO_COOKIE"

	TMP_DOMAIN=$DOMAIN
	sed -i -- "s/{{.Domain}}/$TMP_DOMAIN/g" /profile.origin.traffic_ops
	echo "Got domain: $TMP_DOMAIN"

	echo "Importing origin"
	curl -v -k -X POST -H "Cookie: mojolicious=$TMP_TO_COOKIE" -F "filename=profile.origin.traffic_ops" -F "profile_to_import=@/profile.origin.traffic_ops" $TRAFFIC_OPS_URI/profile/doImport

	curl -v -k -X POST -H "Cookie: mojolicious=$TMP_TO_COOKIE" --data-urlencode "division.name=East" $TRAFFIC_OPS_URI/division/create
	TMP_DIVISION_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TMP_TO_COOKIE" $TRAFFIC_OPS_URI/region/add | grep --color=never -oE "<option value=\"[0-9]+\">East</option>" | grep --color=never -oE "[0-9]+")"
	echo "Got division ID: $TMP_DIVISION_ID"

	curl -v -k -X POST -H "Cookie: mojolicious=$TMP_TO_COOKIE" --data-urlencode "region.name=Eastish" --data-urlencode "region.division_id=$TMP_DIVISION_ID" $TRAFFIC_OPS_URI/region/create
	TMP_REGION_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TMP_TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/regions.json | python -c 'import json,sys;obj=json.load(sys.stdin);match=[x["id"] for x in obj["response"] if x["name"]=="Eastish"]; print match[0]')"
	echo "Got region ID: $TMP_REGION_ID"

	TMP_CACHEGROUP_TYPE="$(curl -s -k -X GET -H "Cookie: mojolicious=$TMP_TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/types.json | python -c 'import json,sys;obj=json.load(sys.stdin);match=[x["id"] for x in obj["response"] if x["name"]=="MID_LOC"]; print match[0]')"
	echo "Got cachegroup type ID: $TMP_CACHEGROUP_TYPE"

	curl -v -k -X POST -H "Cookie: mojolicious=$TMP_TO_COOKIE" --data-urlencode "cg_data.name=mid-east" --data-urlencode "cg_data.short_name=east" --data-urlencode "cg_data.latitude=0" --data-urlencode "cg_data.longitude=0" --data-urlencode "cg_data.parent_cachegroup_id=-1" --data-urlencode "cg_data.type=$TMP_CACHEGROUP_TYPE" $TRAFFIC_OPS_URI/cachegroup/create
	TMP_CACHEGROUP_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TMP_TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/cachegroups.json | python -c 'import json,sys;obj=json.load(sys.stdin);match=[x["id"] for x in obj["response"] if x["name"]=="mid-east"]; print match[0]')"
	echo "Got cachegroup ID: $TMP_CACHEGROUP_ID"

	TMP_CACHEGROUP_EDGE_TYPE="$(curl -s -k -X GET -H "Cookie: mojolicious=$TMP_TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/types.json | python -c 'import json,sys;obj=json.load(sys.stdin);match=[x["id"] for x in obj["response"] if x["name"]=="EDGE_LOC"]; print match[0]')"
	echo "Got cachegroup type ID: $TMP_CACHEGROUP_EDGE_TYPE"

	curl -v -k -X POST -H "Cookie: mojolicious=$TMP_TO_COOKIE" --data-urlencode "cg_data.name=edge-east" --data-urlencode "cg_data.short_name=eeast" --data-urlencode "cg_data.latitude=0" --data-urlencode "cg_data.longitude=0" --data-urlencode "cg_data.parent_cachegroup_id=$TMP_CACHEGROUP_ID" --data-urlencode "cg_data.type=$TMP_CACHEGROUP_EDGE_TYPE" $TRAFFIC_OPS_URI/cachegroup/create
	TMP_CACHEGROUP_EDGE_ID="$(curl -s -k -X GET -H "Cookie: mojolicious=$TMP_TO_COOKIE" $TRAFFIC_OPS_URI/api/1.2/cachegroups.json | python -c 'import json,sys;obj=json.load(sys.stdin);match=[x["id"] for x in obj["response"] if x["name"]=="edge-east"]; print match[0]')"
	echo "Got cachegroup edge ID: $TMP_CACHEGROUP_EDGE_ID"

	curl -v -k -X POST -H "Cookie: mojolicious=$TMP_TO_COOKIE" --data-urlencode "location.name=plocation-nyc-1" --data-urlencode "location.short_name=nyc" --data-urlencode "location.address=1 Main Street" --data-urlencode "location.city=nyc" --data-urlencode "location.state=NY" --data-urlencode "location.zip=12345" --data-urlencode "location.poc=" --data-urlencode "location.phone=" --data-urlencode "location.email=no@no.no" --data-urlencode "location.comments=" --data-urlencode "location.region=$TMP_REGION_ID" $TRAFFIC_OPS_URI/phys_location/create

	if [ -z "$DROP_UNIQUE_IP" ]; then
		# This makes it possible to add multiple servers with the same IP on different ports, which is especially useful for Docker setups.
		# TODO remove when TO is fixed to permit multiple servers with the same hostname and profile on different ports
		PGPASSWORD=${DB_USER_PASS} psql -h ${DB_SERVER} -U ${DB_USER} -c "DROP INDEX IF EXISTS idx_140441_ip_profile"
		PGPASSWORD=${DB_USER_PASS} psql -h ${DB_SERVER} -U ${DB_USER} -c "DROP INDEX IF EXISTS idx_140441_ip6_profile"
	fi

	echo "INITIALIZED=1" >> /etc/environment
}

source /etc/environment
if [ -z "$INITIALIZED" ]; then init; fi
start
