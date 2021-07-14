<!--
    Licensed to the Apache Software Foundation (ASF) under one
    or more contributor license agreements.  See the NOTICE file
    distributed with this work for additional information
    regarding copyright ownership.  The ASF licenses this file
    to you under the Apache License, Version 2.0 (the
    "License"); you may not use this file except in compliance
    with the License.  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing,
    software distributed under the License is distributed on an
    "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
    KIND, either express or implied.  See the License for the
    specific language governing permissions and limitations
    under the License.
-->

# Monitor Load Test

This Docker Compose script creates set of Docker containers designed for load testing the Monitor.

Specifically, it starts:

- a Postgres database for Traffic Ops
- Traffic Ops
- Traffic Monitor

It is intended to be run with a Traffic Ops database dump, with existing servers, which will then be polled by the Traffic Monitor container.

# Running

To run, set the appropriate variables for your configuration in the `.env` file in this directory.

Variables which typically need changed:

- DB_SQL_PATH
  - the path to the SQL file to load into Traffic Ops. E.g. `/Users/you/lang/go/src/github.com/apache/incubator-trafficcontrol/infrastructure/docker/traffic_ops_database/db
- DB_SQL_FILE
  - the file name of the SQL file to load into Traffic Ops, located at DB_SQL_PATH. E.g. `traffic-ops.sql`
  - this file should be dumped from a valid Traffic Ops Postgres database, with e.g. `pg_dump traffic_ops > traffic-ops.sql`
- TO_RPM
  - the filename of the Traffic Ops RPM to build the Docker image with. e.g. `traffic_ops-2.2.0-7354.0f5c6382.el7.x86_64.rpm`
  - this RPM should be built however you build Traffic Ops RPMs, e.g. with `cd $GOPATH/src/github.com/apache/incubator-trafficcontrol/infrastructure/docker/build && docker-compose up traffic_ops_build`
  - this RPM file should be in the directory of the Traffic Ops Dockerfile, e.g. `$GOPATH/src/github.com/apache/incubator-trafficcontrol/infrastructure/docker/traffic_ops`
- TM_RPM
  - the filename of the Traffic Monitor RPM to build the Docker image with. e.g. `traffic_monitor-2.2.0-7354.0f5c6382.el7.x86_64.rpm`
  - this RPM should be built however you build Traffic Ops RPMs, e.g. with `cd $GOPATH/src/github.com/apache/incubator-trafficcontrol/infrastructure/docker/build && docker-compose up traffic_monitor_build`
  - this RPM file should be in the directory of the Traffic Ops Dockerfile, e.g. `$GOPATH/src/github.com/apache/incubator-trafficcontrol/infrastructure/docker/traffic_monitor`
- TM_CACHEGROUP
  - The cachegroup to be assigned to the Monitor, in Traffic Ops, from the DB_SQL_FILE dump. e.g. `my-monitor-cachegroup-name`
- TM_PROFILE
  - The profile to be assigned to the Monitor, in Traffic Ops, from the DB_SQL_FILE dump. e.g. `RASCAL_MYCDN`
- TM_PHYS_LOCATION
  - The physical location to be assigned to the Monitor, in Traffic Ops, from the DB_SQL_FILE dump. Note this must be the short name, not the human-readable physical location name. e.g. `plocation-nyc-1`
- TM_CDN
  - The CDN to be assigned to the Monitor, in Traffic Ops, from the DB_SQL_FILE dump. e.g. `my-cdn-name`

With all the necessary variables specified in the `.env` file, running, from this directory, `docker-compose up` should create the images and run the containers. The containers should populate the Postgres database with the given SQL file, insert the monitor server into Traffic Ops, and start the database, Traffic Ops, and Traffic Monitor, exposing Traffic Ops on `https://localhost` and Traffic Monitor on `http://localhost`.

If everything worked, the Monitor should begin polling the servers in Traffic Ops from the given SQL file.
