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

ARG RHEL_VERSION=8
FROM golang:1.15.5 as builder

WORKDIR /go/src/github.com/apache
RUN git clone https://github.com/apache/trafficcontrol.git --branch 4.1.x --single-branch
WORKDIR trafficcontrol/traffic_stats
RUN go get ./...
RUN go build .


FROM centos:$RHEL_VERSION
ARG RHEL_VERSION=8

RUN /bin/echo "RHEL VERSION: $RHEL_VERSION"

RUN if [[ "${RHEL_VERSION%%.*}" -eq 7 ]]; then \
        yum -y install dnf || exit 1; \
    fi

#WORKDIR /root/
COPY --from=builder /go/bin/traffic_stats /opt/traffic_stats/bin/

RUN mkdir -p /opt/traffic_stats/var/run && mkdir -p /opt/traffic_stats/var/log && mkdir -p /opt/traffic_stats/conf

ADD run.sh /
ENTRYPOINT /run.sh
#CMD ["./traffic_stats","-cfg","/opt/traffic_stats/conf/traffic_stats.cfg"]
