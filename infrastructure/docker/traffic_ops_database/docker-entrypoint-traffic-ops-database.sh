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

envvars=(POSTGRES_USER POSTGRES_PASSWORD TODB_USERNAME TODB_NAME TODB_USERNAME_PASSWORD)
for v in $envvars
do
	if [[ -z $$v ]]; then echo "$v is unset"; exit 1; fi
done

# Executes the given argument as a command, every second, until they succeed, up to retrys seconds. If retrys elapses, and there is no success, this exits the script with a nonzero exit code.
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

docker-entrypoint.sh $@ &
retry pg_isready # note this only gets us to the first start; docker-entrypoint.sh stops and starts again, hence we must retry actual psql commands

start() {
	tail -f /dev/null
}

init() {
	# because the Postgres Dockerfile starts, then stops, then starts again, there's no way inside the container to ensure commands succeed, except retrying. We can't even know Postgres is permanently up after one succeeds.
	retry psql -U $POSTGRES_USER -c "CREATE USER $TODB_USERNAME WITH ENCRYPTED PASSWORD '$TODB_USERNAME_PASSWORD'"
	retry psql -U $POSTGRES_USER -c "CREATE DATABASE $TODB_NAME OWNER $TODB_USERNAME"
	if [[ ! -z $DB_SQL_PATH && -f $DB_SQL_PATH ]]; then
		retry psql -U $POSTGRES_USER -d $TODB_NAME -1 -f $DB_SQL_PATH

		if [[ ! -z $TO_USER && ! -z $TO_PASS ]]; then
			echo "DEBUGQ TO_PASS: ${TO_PASS}"
			TO_PASS_HASH=$(perl -MCrypt::ScryptKDF=scrypt_hash -e "chomp; print scrypt_hash($TO_PASS, \64, 16384, 8, 1, 64);")
			echo $?
			echo "DEBUGQ pass_hash: X${TO_PASS_HASH}X"
			retry psql -U $POSTGRES_USER -d $TODB_NAME -c "INSERT INTO tm_user (username, local_passwd, role) VALUES ('$TO_USER', '$TO_PASS_HASH', (select id from role where name = 'admin'))"
		fi
	fi
	echo "INITIALIZED=1" >> /etc/environment
	echo "INITIALIZED"
}

source /etc/environment
if [ -z "$INITIALIZED" ]; then init; fi
start
