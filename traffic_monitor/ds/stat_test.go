package ds

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
	"errors"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_monitor/cache"
	"github.com/apache/trafficcontrol/traffic_monitor/dsdata"
	"github.com/apache/trafficcontrol/traffic_monitor/health"
	"github.com/apache/trafficcontrol/traffic_monitor/peer"
	"github.com/apache/trafficcontrol/traffic_monitor/threadsafe"
	"github.com/apache/trafficcontrol/traffic_monitor/todata"
)

func TestCreateStats(t *testing.T) {
	toData := getMockTOData()
	combinedCRStates := peer.NewCRStatesThreadsafe()
	lastStatsThs := threadsafe.NewLastStats()
	now := time.Now()
	maxEvents := uint64(4)
	events := health.NewThreadsafeEvents(maxEvents)
	localCRStates := peer.NewCRStatesThreadsafe()

	dses := []string}
	for ds, _ := range toData.DeliveryServiceServers {
		dses = append(dses, ds)
	}

	caches := []string}
	for cache, _ := range toData.ServerDeliveryServices {
		caches = append(caches, cache)
	}

	for _, cache := range caches {
		combinedCRStates.AddCache(cache, tc.IsAvailable{IsAvailable: true})
		localCRStates.AddCache(cache, tc.IsAvailable{IsAvailable: true})
	}

	precomputeds := randCachesPrecomputedData(caches, toData)

	monitorConfig := getMockMonitorConfig(dses)

	lastStatsVal := lastStatsThs.Get()
	lastStatsCopy := lastStatsVal.Copy()
	dsStats, err := CreateStats(precomputeds, toData, combinedCRStates.Get(), lastStatsCopy, now, monitorConfig, events, localCRStates)

	if err != nil {
		t.Fatalf("CreateStats err expected: nil, actual: " + err.Error())
	}

	lastStatsThs.Set(*lastStatsCopy)

	cgMap := map[tc.CacheGroupName]struct{}{}
	for _, cg := range toData.ServerCachegroups {
		cgMap[cg] = struct{}{}
	}

	tpMap := map[tc.CacheType]struct{}{}
	for _, tp := range toData.ServerTypes {
		tpMap[tp] = struct{}{}
	}

	caMap := map[string]struct{}{}
	for ca, _ := range toData.ServerDeliveryServices {
		caMap[ca] = struct{}{}
	}

	for dsName, dsStat := range dsStats.DeliveryService {
		for cgName, cgStat := range dsStat.CacheGroups {
			if _, ok := cgMap[cgName]; !ok {
				t.Fatalf("CreateStats cachegroup expected: %+v, actual: %+v", cgMap, cgName)
			}

			cgExpected := cache.AStat{}
			for pCache, pData := range precomputeds {
				if toData.ServerCachegroups[pCache] != cgName {
					continue
				}

				if pDataDS, ok := pData.DeliveryServiceStats[dsName]; ok {
					cgExpected.InBytes += pDataDS.InBytes
					cgExpected.OutBytes += pDataDS.OutBytes
					cgExpected.Status2xx += pDataDS.Status2xx
					cgExpected.Status3xx += pDataDS.Status3xx
					cgExpected.Status4xx += pDataDS.Status4xx
					cgExpected.Status5xx += pDataDS.Status5xx
				}
			}

			if errStr := compareAStatToStatCacheStats(&cgExpected, cgStat); errStr != "" {
				t.Fatalf("CreateStats cachegroup " + string(cgName) + ": " + errStr)
			}

		}

		for tpName, tpStat := range dsStat.Types {
			if _, ok := tpMap[tpName]; !ok {
				t.Fatalf("CreateStats type expected: %+v, actual: %+v", tpMap, tpName)
			}

			tpExpected := cache.AStat{}
			for pCache, pData := range precomputeds {
				if toData.ServerTypes[pCache] != tpName {
					continue
				}

				if pDataDS, ok := pData.DeliveryServiceStats[dsName]; ok {
					tpExpected.InBytes += pDataDS.InBytes
					tpExpected.OutBytes += pDataDS.OutBytes
					tpExpected.Status2xx += pDataDS.Status2xx
					tpExpected.Status3xx += pDataDS.Status3xx
					tpExpected.Status4xx += pDataDS.Status4xx
					tpExpected.Status5xx += pDataDS.Status5xx
				}
			}

			if errStr := compareAStatToStatCacheStats(&tpExpected, tpStat); errStr != "" {
				t.Fatalf("CreateStats type " + string(tpName) + ": " + errStr)
			}
		}

		for caName, caStat := range dsStat.Caches {
			if _, ok := caMap[caName]; !ok {
				t.Fatalf("CreateStats cache expected: %+v, actual: %+v", caMap, caName)
			}

			caExpected := cache.AStat{}
			for pCache, pData := range precomputeds {
				if pCache != caName {
					continue
				}

				if pDataDS, ok := pData.DeliveryServiceStats[dsName]; ok {
					caExpected.InBytes += pDataDS.InBytes
					caExpected.OutBytes += pDataDS.OutBytes
					caExpected.Status2xx += pDataDS.Status2xx
					caExpected.Status3xx += pDataDS.Status3xx
					caExpected.Status4xx += pDataDS.Status4xx
					caExpected.Status5xx += pDataDS.Status5xx
				}
			}

			if errStr := compareAStatToStatCacheStats(&caExpected, caStat); errStr != "" {
				t.Fatalf("CreateStats cache " + string(caName) + ": " + errStr)
			}
		}

		{
			cmStat := dsStat.CommonStats

			if int(cmStat.CachesConfiguredNum.Value) != len(toData.DeliveryServiceServers[dsName]) {
				t.Fatalf("CreateStats CommonStats.CachesConfiguredNum expected: %+v actual: %+v", len(toData.DeliveryServiceServers[dsName]), dsStat.CommonStats.CachesConfiguredNum.Value)
			}

			for caName, reporting := range cmStat.CachesReporting {
				if _, ok := caMap[caName]; !ok {
					t.Fatalf("CreateStats CommonStats.CachesReporting '%+v' not in test caches", caName)
				}
				if !reporting {
					t.Fatalf("CreateStats len(CommonStats.CachesReporting[%+v] expected: true actual: false", caName)
				}
			}

			if cmStat.ErrorStr.Value != "" {
				t.Fatalf("CreateStats CommonStats.ErrorStr expected: '' actual: %+v", cmStat.ErrorStr.Value)
			}

			if cmStat.StatusStr.Value != "" {
				t.Fatalf("CreateStats CommonStats.StatusStr expected: '' actual: '%+v'", cmStat.StatusStr.Value)
			}
		}
	}

	if len(lastStatsCopy.DeliveryServices) != len(toData.DeliveryServiceServers) {
		t.Fatalf("CreateStats len(LastStats.DeliveryServices) expected: %+v actual: %+v", len(toData.DeliveryServiceServers), len(lastStatsCopy.DeliveryServices))
	}

	if len(lastStatsCopy.Caches) != len(toData.ServerDeliveryServices) {
		t.Fatalf("CreateStats len(LastStats.Caches) expected: %+v actual: %+v", len(toData.ServerDeliveryServices), len(lastStatsCopy.Caches))
	}

}

// compareAStatToStatCacheStats compares the two stats, and returns an error string, which is empty of both are equal.
// The fields in StatCacheStats but not AStat are ignored.
func compareAStatToStatCacheStats(expected *cache.AStat, actual *dsdata.StatCacheStats) string {
	if actual.InBytes.Value != float64(expected.InBytes) {
		return fmt.Sprintf("InBytes expected: \n%+v, actual: \n%+v", expected.InBytes, actual.InBytes.Value)
	}

	if actual.OutBytes.Value != int64(expected.OutBytes) {
		return fmt.Sprintf("OutBytes expected: \n%+v, actual: \n%+v", expected.OutBytes, actual.OutBytes.Value)
	}

	if actual.Status2xx.Value != int64(expected.Status2xx) {
		return fmt.Sprintf("Status2xx expected: \n%+v, actual: \n%+v", expected.Status2xx, actual.Status2xx.Value)
	}

	if actual.Status3xx.Value != int64(expected.Status3xx) {
		return fmt.Sprintf("Status3xx expected: \n%+v, actual: \n%+v", expected.Status3xx, actual.Status3xx.Value)
	}

	if actual.Status4xx.Value != int64(expected.Status4xx) {
		return fmt.Sprintf("Status4xx expected: \n%+v, actual: \n%+v", expected.Status4xx, actual.Status4xx.Value)
	}

	if actual.Status5xx.Value != int64(expected.Status5xx) {
		return fmt.Sprintf("Status5xx expected: \n%+v, actual: \n%+v", expected.Status5xx, actual.Status5xx.Value)
	}

	if actual.ErrorString.Value != "" {
		return fmt.Sprintf("ErrorString expected: empty, actual: %+v", actual.ErrorString.Value)
	}

	return ""
}

func getMockMonitorDSNoThresholds(name string) tc.TMDeliveryService {
	return tc.TMDeliveryService{
		XMLID:              name,
		TotalTPSThreshold:  math.MaxInt64,
		ServerStatus:       string(tc.CacheStatusReported),
		TotalKbpsThreshold: math.MaxInt64,
	}
}

func getMockMonitorDSLowThresholds(name string) tc.TMDeliveryService {
	return tc.TMDeliveryService{
		XMLID:              name,
		TotalTPSThreshold:  1,
		ServerStatus:       string(tc.CacheStatusReported),
		TotalKbpsThreshold: 1,
	}
}

func getMockMonitorConfig(dses []string) tc.TrafficMonitorConfigMap {
	mc := tc.TrafficMonitorConfigMap{
		TrafficServer:   map[string]tc.TrafficServer{},
		CacheGroup:      map[string]tc.TMCacheGroup{},
		Config:          map[string]interface{}{},
		TrafficMonitor:  map[string]tc.TrafficMonitor{},
		DeliveryService: map[string]tc.TMDeliveryService{},
		Profile:         map[string]tc.TMProfile{},
	}

	tmDSes := map[string]tc.TMDeliveryService{}
	for _, ds := range dses {
		tmDSes[ds] = getMockMonitorDSNoThresholds(ds)
	}
	mc.DeliveryService = tmDSes

	return mc
}

func getMockTOData() todata.TOData {
	numCaches := 100
	numDSes := 100
	numCacheDSes := numDSes / 3
	numCGs := 20

	types := []tc.CacheType{tc.CacheTypeEdge, tc.CacheTypeEdge, tc.CacheTypeEdge, tc.CacheTypeEdge, tc.CacheTypeEdge, tc.CacheTypeMid}

	caches := []string}
	for i := 0; i < numCaches; i++ {
		caches = append(caches, randStr())
	}

	dses := []string}
	for i := 0; i < numDSes; i++ {
		dses = append(dses, randStr())
	}

	cgs := []tc.CacheGroupName{}
	for i := 0; i < numCGs; i++ {
		cgs = append(cgs, tc.CacheGroupName(randStr()))
	}

	serverDSes := map[string][]string}
	for _, ca := range caches {
		for i := 0; i < numCacheDSes; i++ {
			serverDSes[ca] = append(serverDSes[ca], dses[rand.Intn(len(dses))])
		}
	}

	dsServers := map[string][]string}
	for server, dses := range serverDSes {
		for _, ds := range dses {
			dsServers[ds] = append(dsServers[ds], server)
		}
	}

	serverCGs := map[string]tc.CacheGroupName{}
	for _, cache := range caches {
		serverCGs[cache] = cgs[rand.Intn(len(cgs))]
	}

	serverTypes := map[string]tc.CacheType{}
	for _, cache := range caches {
		serverTypes[cache] = types[rand.Intn(len(types))]
	}

	tod := todata.New()
	tod.DeliveryServiceServers = dsServers
	tod.ServerDeliveryServices = serverDSes
	tod.ServerTypes = serverTypes
	tod.ServerCachegroups = serverCGs
	return *tod
}

func randCachesPrecomputedData(caches []string toData todata.TOData) map[string]cache.PrecomputedData {
	prc := map[string]cache.PrecomputedData{}
	for _, ca := range caches {
		prc[ca] = randPrecomputedData(toData)
	}
	return prc
}

func randPrecomputedData(toData todata.TOData) cache.PrecomputedData {
	dsStats := randDsStats(toData)
	dsTotal := uint64(0)
	for _, stat := range dsStats {
		dsTotal += stat.OutBytes
	}
	return cache.PrecomputedData{
		DeliveryServiceStats: dsStats,
		OutBytes:             int64(dsTotal),
		MaxKbps:              rand.Int63(),
		Errors:               randErrs(),
		Reporting:            true,
	}
}

func randDsStats(toData todata.TOData) map[string]*cache.AStat {
	a := map[string]*cache.AStat{}
	for ds, _ := range toData.DeliveryServiceServers {
		a[ds] = randAStat()
	}
	return a
}

func randAStat() *cache.AStat {
	return &cache.AStat{
		InBytes:   uint64(rand.Intn(1000)),
		OutBytes:  uint64(rand.Intn(1000)),
		Status2xx: uint64(rand.Intn(1000)),
		Status3xx: uint64(rand.Intn(1000)),
		Status4xx: uint64(rand.Intn(1000)),
		Status5xx: uint64(rand.Intn(1000)),
	}
}

func randErrs() []error {
	if randBool() {
		return []error{}
	}
	num := 5
	errs := []error{}
	for i := 0; i < num; i++ {
		errs = append(errs, errors.New(randStr()))
	}
	return errs
}

func randBool() bool {
	return rand.Int()%2 == 0
}

func randStr() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-_"
	num := 100
	s := ""
	for i := 0; i < num; i++ {
		s += string(chars[rand.Intn(len(chars))])
	}
	return s
}

func TestAddLastStatsToStatCacheStatsNilVals(t *testing.T) {
	// test that addLastStatsToStatCacheStats doesn't panic with nil values
	addLastStatsToStatCacheStats(nil, nil)
	addLastStatsToStatCacheStats(&dsdata.StatCacheStats{}, nil)
	addLastStatsToStatCacheStats(nil, &dsdata.LastStatsData{})
}
