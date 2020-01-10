package cache

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

// stats_type_noop is a no-op parser designed to work with the the noop poller, to report caches as healthy without actually polling them.

import (
	"github.com/apache/trafficcontrol/lib/go-tc/enum"
	"io"

	"github.com/apache/trafficcontrol/traffic_monitor/todata"
)

const StatsTypeNOOP = "noop"

func init() {
	AddStatsType(StatsTypeNOOP, noopParse, noopPrecompute)
}

func noopParse(cache enum.CacheName, r io.Reader) (error, map[string]interface{}, AstatsSystem) {
	// we need to make a fake system, so the health parse succeeds
	return nil, map[string]interface{}{}, AstatsSystem{
		ProcLoadavg: "0.10 0.05 0.05 1/1000 30000",
		ProcNetDev:  "bond0:10000 10000    0    0    0     0          0   1000 100000 1000000    0    0    0     0       0          0",
		InfSpeed:    20000,
		InfName:     "bond0",
	}
}

func noopPrecompute(cache enum.CacheName, toData todata.TOData, rawStats map[string]interface{}, system AstatsSystem) PrecomputedData {
	return PrecomputedData{DeliveryServiceStats: map[enum.DeliveryServiceName]*AStat{}}
}
