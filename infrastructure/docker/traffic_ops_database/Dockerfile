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
# Dockerfile to create a database for Traffic Ops container images
############################################################

# This Dockerfile creates a Postgres database container images for Traffic Ops.
# When the container is run, if a volume is mounted containing a SQL file with an initial database, and the environment variable DB_SQL_PATH is set to the path of the SQL file, the container will populate the database with that file.
#
# Example, where `./db` on the host contains the file `db.sql`:
# docker run -it --name traffic_ops_database --hostname traffic_ops_database --net cdnet --env POSTGRES_USER=postgres --env POSTGRES_PASSWORD=secretrootpass --env TODB_USERNAME=traffic_ops --env TODB_NAME=traffic_ops --env TODB_USERNAME_PASSWORD=supersecretpassword --env DB_SQL_PATH=/db/db.sql -v ./db:/db -- traffic_ops_database
#
# When the container is run, if the environment variables TO_USER and TO_PASS are set, that user and password will be inserted into the database, after any SQL file is loaded.
# This allows you to load a real-world database into Docker for testing, in which all available users use LDAP, creating a user with a local password in order to log in without configuring LDAP in the Docker container.

FROM postgres:9.6
RUN apt update
RUN apt install -y gcc tar curl make
# RUN curl -L https://cpanmin.us | perl - --sudo App::cpanminus
RUN perl -MCPAN -e 'my $c = "CPAN::HandleConfig"; $c->load(doit => 1, autoconfig => 1); $c->edit(prerequisites_policy => "follow"); $c->edit(build_requires_install_policy => "yes"); $c->commit'
RUN cpan install App::cpanminus
# RUN cpan o conf prerequisites_policy follow
# RUN cpan o conf commit
RUN cpan Crypt::PRNG
RUN cpan Bytes::Random::Secure
RUN cpan Crypt::ScryptKDF

COPY docker-entrypoint-traffic-ops-database.sh /usr/local/bin/
ENTRYPOINT ["docker-entrypoint-traffic-ops-database.sh"]
EXPOSE 5432
CMD ["postgres"]
