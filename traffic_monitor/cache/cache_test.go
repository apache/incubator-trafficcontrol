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

import (
	"github.com/apache/trafficcontrol/lib/go-tc/enum"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_monitor/todata"
)

func TestHandlerPrecompute(t *testing.T) {
	if NewHandler().Precompute() {
		t.Errorf("expected NewHandler().Precompute() false, actual true")
	}
	if !NewPrecomputeHandler(todata.NewThreadsafe()).Precompute() {
		t.Errorf("expected NewPrecomputeHandler().Precompute() true, actual false")
	}
}

type DummyFilterNever struct {
}

func (f DummyFilterNever) UseStat(name string) bool {
	return false
}

func (f DummyFilterNever) UseCache(name enum.CacheName) bool {
	return false
}

func (f DummyFilterNever) WithinStatHistoryMax(i int) bool {
	return false
}

func TestComputeStatGbps(t *testing.T) {
	serverInfo := tc.TrafficServer{}
	serverProfile := tc.TMProfile{}
	combinedState := tc.IsAvailable{}
	computedStats := ComputedStats()
	got := computedStats["gbps"](ResultInfo{Vitals: Vitals{KbpsOut: 1500000}}, serverInfo, serverProfile, combinedState)
	want := 1.5
	if got != want {
		t.Errorf("ComputedStats[\"gbps\"] return %v instead of %v", got, want)
	}

	got = computedStats["gbps"](ResultInfo{Vitals: Vitals{KbpsOut: 1400000}}, serverInfo, serverProfile, combinedState)
	want = 1.4
	if got != want {
		t.Errorf("ComputedStats[\"gbps\"] return %v instead of %v", got, want)
	}
}
