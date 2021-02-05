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
# The following environment variables are used to configure the database and traffic ops.  They must be set
# in ../variables.env for docker-compose to pick up the values:
#
# DB_SERVER
# DB_PORT
# DB_USER
# DB_USER_PASS
# DB_NAME
# ADMIN_USER
# ADMIN_PASS
# TODO:  Unused -- should be removed?  TRAFFIC_VAULT_PASS

set -ex

# Check that env vars are set
envvars=( DB_SERVER DB_PORT DB_USER DB_USER_PASS ADMIN_USER ADMIN_PASS X509_CA_DIR TLD_DOMAIN INFRA_SUBDOMAIN CDN_SUBDOMAIN DS_HOSTS)
for v in $envvars
do
	if [[ -z "${!v}" ]]; then echo "$v is unset"; exit 1; fi
done

set-dns.sh
insert-self-into-dns.sh

set-to-ips-from-dns.sh

# Source to-access functions and FQDN vars
source /to-access.sh

# Create SSL certificates and trust the shared CA.
source /generate-certs.sh

# copy contents of /ca to /export/ssl
# update the permissions
mkdir -p "$X509_CA_PERSIST_DIR" && chmod 777 "$X509_CA_PERSIST_DIR"
chmod -R a+r "$X509_CA_PERSIST_DIR"

if [ -r "$X509_CA_PERSIST_ENV_FILE" ] ; then
	umask $X509_CA_UMASK
	mkdir -p "$X509_CA_DIR" && chmod 777 $X509_CA_DIR
	rsync -a "$X509_CA_PERSIST_DIR/" "$X509_CA_DIR/"
	sync
	echo "PERSIST CERTS FROM $X509_CA_PERSIST_DIR to $X509_CA_DIR"
	sleep 4
	source "$X509_CA_ENV_FILE"
elif x509v3_init; then
	umask $X509_CA_UMASK
	x509v3_create_cert "$INFRA_SUBDOMAIN" "$INFRA_FQDN"
	for ds in $DS_HOSTS
	do
		x509v3_create_cert "$ds" "$ds.$CDN_FQDN"
	done
	echo "X509_GENERATION_COMPLETE=\"YES\"" >> "$X509_CA_ENV_FILE"
	x509v3_dump_env
	# Save newly generated certs for future restarts.
	rsync -av "$X509_CA_DIR/" "$X509_CA_PERSIST_DIR/"
	chmod -R a+r "$X509_CA_DIR/" "$X509_CA_PERSIST_DIR"
	sync
	echo "GENERATE CERTS FROM $X509_CA_DIR to $X509_CA_PERSIST_DIR"
	sleep 4
fi

chown -R trafops:trafops "$X509_CA_PERSIST_DIR"
chmod -R a+r "$X509_CA_PERSIST_DIR"

# Write config files
set -x
if [[ -r /config.sh ]]; then
	. /config.sh
fi

pg_isready=$(rpm -ql postgresql96 | grep bin/pg_isready)
if [[ ! -x $pg_isready ]] ; then
	echo "Can't find pg_ready in postgresql96"
	exit 1
fi

while ! $pg_isready -h$DB_SERVER -p$DB_PORT -d $DB_NAME; do
	echo "waiting for db on $DB_SERVER $DB_PORT"
	sleep 3
done

export TO_DIR=/opt/traffic_ops/app
cat conf/production/database.conf

export PERL5LIB=$TO_DIR/lib:$TO_DIR/local/lib/perl5
export PATH=/usr/local/go/bin:/opt/traffic_ops/go/bin:$PATH
export GOPATH=/opt/traffic_ops/go

cd "$TO_DIR/db"
./admin --env=production --use-sqitch --db-path . reset
./admin --env=production --use-sqitch --db-path . upgrade

# Add admin user -- all other users should be created using API
/adduser.pl $TO_ADMIN_USER $TO_ADMIN_PASSWORD admin | psql -v ON_ERROR_STOP=1 -U$DB_USER -h$DB_SERVER $DB_NAME

cd "$TO_DIR";

touch "$TO_LOG_ERROR" "$TO_LOG_WARNING" "$TO_LOG_INFO" "$TO_LOG_DEBUG" "$TO_LOG_EVENT" /var/log/traffic_ops/perl_access.log
tail -qf "$TO_LOG_ERROR" "$TO_LOG_WARNING" "$TO_LOG_INFO" "$TO_LOG_DEBUG" "$TO_LOG_EVENT" /var/log/traffic_ops/perl_access.log &

if [[ "$TO_PERL_DEBUG_ENABLE" == true ]]; then
	set -o allexport;
	PERL5_DEBUG_HOST=0.0.0.0;
	PERL5_DEBUG_PORT=5000;
	PERL5_DEBUG_ROLE=server;
	MOJO_LISTEN="${TO_PERL_SCHEME}://*:${TO_PERL_PORT}?cert=${X509_CA_DIR}/${INFRA_FQDN}.crt&key=${X509_CA_DIR}/${INFRA_FQDN}.key&verify=0x00&ciphers=AES128-GCM-SHA256:HIGH:"'!RC4:!MD5:!aNULL:!EDH:!ED';
	MOJO_INACTIVITY_TIMEOUT=$(( 60 * 60 * 24 )); # 24 hours
	set +o allexport;

	perl -d:Camelcadedb $TO_DIR/local/bin/morbo script/cdn &
else
	$TO_DIR/local/bin/hypnotoad script/cdn &
fi;

until [[ -f ${ENROLLER_DIR}/enroller-started ]]; do
	echo "waiting for enroller"
	sleep 3
done

# Add initial data to traffic ops
/trafficops-init.sh

export TO_USER=$TO_ADMIN_USER
export TO_PASSWORD=$TO_ADMIN_PASSWORD

to-enroll "to" ALL || (while true; do echo "enroll failed."; sleep 3 ; done)

exec tail -f /dev/null
