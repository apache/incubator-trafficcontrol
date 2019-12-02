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

# Check for proper invocation
PWD := $(strip $(lastword $(patsubst %/,%,$(notdir $(shell pwd)))))
makefile_dir := $(strip $(notdir $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))))

ifneq ($(PWD),$(makefile_dir))
$(error This makefile MUST be run from within its directory)
endif

BUILD_NUMBER := $(shell git rev-list HEAD 2>/dev/null | wc -l | tr -d '[[:space:]]').$(shell git rev-parse --short=8 HEAD)
TC_VERSION := $(shell cat "../../VERSION")
TOMCAT_VERSION := $(shell grep 'export TOMCAT_VERSION=' ../../traffic_router/build/build_rpm.sh  | cut -d '=' -f 2)
TOMCAT_RELEASE := $(shell grep 'export TOMCAT_RELEASE=' ../../traffic_router/build/build_rpm.sh  | cut -d '=' -f 2)

RHEL_VERSION := el7

SPECIAL_SAUCE := $(TC_VERSION)-$(BUILD_NUMBER).$(RHEL_VERSION).x86_64.rpm
SPECIAL_SEASONING := $(TOMCAT_VERSION).$(TOMCAT_RELEASE)-$(BUILD_NUMBER).$(RHEL_VERSION).x86_64.rpm

TO_SOURCE := $(wildcard ../../traffic_ops/**/*)
TO_SOURCE += $(wildcard ../../traffic_ops_db/**/*)
TM_SOURCE := $(wildcard ../../traffic_monitor/**/*)
TP_SOURCE := $(wildcard ../../traffic_portal/**/*)
TR_SOURCE := $(wildcard ../../traffic_router/**/*)
TS_SOURCE := $(wildcard ../../traffic_stats/**/*)

.PHONY: clean very-clean all nearly-all

# Default target; builds all pre-requisite rpms from source trees
all: traffic_monitor/traffic_monitor.rpm traffic_portal/traffic_portal.rpm traffic_ops/traffic_ops.rpm traffic_router/traffic_router.rpm traffic_router/tomcat.rpm traffic_stats/traffic_stats.rpm

# Actual output rpm recipies
traffic_monitor/traffic_monitor.rpm: ../../dist/traffic_monitor-$(SPECIAL_SAUCE)
	cp -f $? $@
traffic_ops/traffic_ops.rpm: ../../dist/traffic_ops-$(SPECIAL_SAUCE)
	cp -f $? $@
traffic_portal/traffic_portal.rpm: ../../dist/traffic_portal-$(SPECIAL_SAUCE)
	cp -f $? $@
traffic_router/traffic_router.rpm: ../../dist/traffic_router-$(SPECIAL_SAUCE)
	cp -f $? $@
traffic_router/tomcat.rpm: ../../dist/tomcat-$(SPECIAL_SEASONING)
	cp -f $? $@
traffic_stats/traffic_stats.rpm: ../../dist/traffic_stats-$(SPECIAL_SAUCE)
	cp -f $? $@

# Dist rpms
../../dist/traffic_monitor-$(SPECIAL_SAUCE): $(TM_SOURCE)
	sudo ../../pkg -v traffic_monitor_build

../../dist/traffic_ops-$(SPECIAL_SAUCE): $(TO_SOURCE)
	sudo ../../pkg -v traffic_ops_build

../../dist/traffic_portal-$(SPECIAL_SAUCE): $(TP_SOURCE)
	sudo ../../pkg -v traffic_portal_build

../../dist/traffic_rou%er-$(SPECIAL_SAUCE) ../../dist/tomca%-$(SPECIAL_SEASONING): $(TR_SOURCE)
	sudo ../../pkg -v traffic_router_build

../../dist/traffic_stats-$(SPECIAL_SAUCE): $(TS_SOURCE)
	sudo ../../pkg -v traffic_stats_build

clean:
	$(RM) traffic_ops/traffic_ops.rpm traffic_monitor/traffic_monitor.rpm traffic_portal/traffic_portal.rpm traffic_router/traffic_router.rpm traffic_router/tomcat.rpm edge/traffic_ops_ort.rpm mid/traffic_ops_ort.rpm

very-clean: clean
	$(warning This will destroy ALL OUTPUT RPMS IN 'dist'. Please be sure this is what you want)
	sleep 2 # Give users a sceond to cancel
	$(RM) ../../dist/*
