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

# Update system shared CA support
source /to-access.sh

# Wait on SSL certificate generation
until [ -f "$X509_CA_DONE_FILE" ] 
do
     echo "Waiting on Shared SSL certificate generation"
     sleep 3
done

# Source the CIAB-CA shared SSL environment
source "$X509_CA_ENV_FILE"

# Copy the CIAB-CA certificate to the traffic_router conf so it can be added to the trust store
cp $X509_CA_CERT_FILE /usr/local/share/ca-certificates
update-ca-certificates

# Grep out the existing SSL and Socket listener config
cp -af /etc/riak/riak.conf /etc/riak/riak.conf.orig
grep -v -E '^(listener|#)' /etc/riak/riak.conf.orig  | uniq | sort > /etc/riak/riak.conf

# Update the riak listener config
echo "nodename = riak@0.0.0.0" >> /etc/riak.conf
echo "listener.protobuf.internal = 0.0.0.0:$TV_INT_PORT" >> /etc/riak/riak.conf
echo "listener.http.internal = 0.0.0.0:$TV_HTTP_PORT" >> /etc/riak/riak.conf
echo "listener.https.internal = 0.0.0.0:$TV_HTTPS_PORT" >> /etc/riak/riak.conf

# Update SSL/TLS Certificate Config
echo "ssl.certfile = $X509_INFRA_CERT_FILE" >> /etc/riak/riak.conf
echo "ssl.keyfile = $X509_INFRA_KEY_FILE" >> /etc/riak/riak.conf
echo "ssl.cacertfile = /etc/pki/tls/certs/ca-bundle.crt" >> /etc/riak/riak.conf

# Enable search with Apache Solr
echo "search = on" >>  /etc/riak/riak.conf
