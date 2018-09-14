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
############################################################

. ../traffic_ops/to-access.sh

export TO_URL=https://$TO_FQDN:$TO_PORT
export TO_USER=$TO_ADMIN_USER
export TO_PASSWORD=$TO_ADMIN_PASSWORD

# Wait on SSL certificate generation
until [ -f "$CERT_DONE_FILE" ] 
do
     echo "Waiting on Shared SSL certificate generation"
     sleep 3
done

# Source the CIAB-CA shared SSL environment
source "$CERT_ENV_FILE"
 
# Copy the CIAB-CA certificate to the traffic_router conf so it can be added to the trust store
cp $CERT_CA_CERT_FILE /usr/local/share/ca-certificates
update-ca-certificates

# Traffic Ops must be accepting connections before enroller can start
until nc -z $TO_HOST $TO_PORT </dev/null >/dev/null && to-ping; do
    echo "Waiting for $TO_URL"
    sleep 5
done

mkdir -p "$ENROLLER_DIR"
if [[ ! -d $ENROLLER_DIR ]]; then
     echo "enroller dir ${ENROLLER_DIR} not found or not a directory"
     exit 1
fi

# clear out the enroller dir first so no files left from previous run
rm -rf ${ENROLLER_DIR}/*

./enroller "$ENROLLER_DIR" || tail -f /dev/null
