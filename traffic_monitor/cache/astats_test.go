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
	"bytes"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_monitor/poller"
	"github.com/apache/trafficcontrol/traffic_monitor/todata"
)

func TestAstatsJson(t *testing.T) {
	file, err := os.Open("astats.json")
	if err != nil {
		t.Fatal(err)
	}

	pl := &poller.HTTPPollCtx{HTTPHeader: http.Header{}}
	ctx := interface{}(pl)
	ctx.(*poller.HTTPPollCtx).HTTPHeader.Set("Content-Type", "text/json")
	_, thismap, err := astatsParse("testCache", file, ctx)

	if err != nil {
		t.Error(err)
	}

	t.Logf("Found %v key/val pairs in ats\n", len(thismap))

}

func TestAstatsCSV(t *testing.T) {
	file, err := os.Open("astats.csv")
	if err != nil {
		t.Fatal(err)
	}

	pl := &poller.HTTPPollCtx{HTTPHeader: http.Header{}}
	ctx := interface{}(pl)
	ctx.(*poller.HTTPPollCtx).HTTPHeader.Set("Content-Type", "text/csv")
	_, thismap, err := astatsParse("testCache", file, ctx)

	if err != nil {
		t.Error(err)
	}

	t.Logf("Found %v key/val pairs in ats\n", len(thismap))

}

func BenchmarkAstatsJson(b *testing.B) {
	file, err := ioutil.ReadFile("astats.json")
	if err != nil {
		b.Fatal(err)
	}

	pl := &poller.HTTPPollCtx{HTTPHeader: http.Header{}}
	ctx := interface{}(pl)
	ctx.(*poller.HTTPPollCtx).HTTPHeader.Set("Content-Type", "text/json")
	// Reset benchmark timer to not include reading the file
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := astatsParse("testCache", bytes.NewReader(file), ctx)

		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkAstatsCSV(b *testing.B) {
	file, err := ioutil.ReadFile("astats.csv")
	if err != nil {
		b.Fatal(err)
	}

	// Reset benchmark timer to not include reading the file
	b.ResetTimer()
	pl := &poller.HTTPPollCtx{HTTPHeader: http.Header{}}
	ctx := interface{}(pl)
	ctx.(*poller.HTTPPollCtx).HTTPHeader.Set("Content-Type", "text/csv")
	// Reset benchmark timer to not include reading the file
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := astatsParse("testCache", bytes.NewReader(file), ctx)

		if err != nil {
			b.Error(err)
		}
	}
}

func getMockTODataDSNameDirectMatches() map[tc.DeliveryServiceName]string {
	return map[tc.DeliveryServiceName]string{
		"ds0": "ds0.example.invalid",
		"ds1": "ds1.example.invalid",
	}
}

func getMockTOData(dsNameFQDNs map[tc.DeliveryServiceName]string) todata.TOData {
	tod := todata.New()
	for dsName, dsDirectMatch := range dsNameFQDNs {
		tod.DeliveryServiceRegexes.DirectMatches[dsDirectMatch] = dsName
	}
	return *tod
}

func getMockRawStats(cacheName string, dsNameFQDNs map[tc.DeliveryServiceName]string) map[string]interface{} {
	st := map[string]interface{}{}
	for _, dsFQDN := range dsNameFQDNs {
		st["plugin.remap_stats."+dsFQDN+".in_bytes"] = float64(rand.Uint64())
		st["plugin.remap_stats."+dsFQDN+".out_bytes"] = float64(rand.Uint64())
		st["plugin.remap_stats."+dsFQDN+".status_2xx"] = float64(rand.Uint64())
		st["plugin.remap_stats."+dsFQDN+".status_3xx"] = float64(rand.Uint64())
		st["plugin.remap_stats."+dsFQDN+".status_4xx"] = float64(rand.Uint64())
		st["plugin.remap_stats."+dsFQDN+".status_5xx"] = float64(rand.Uint64())
	}
	return st
}

func getMockStatistics(infSpeed int64, outBytes uint64) Statistics {
	infName := randStr()
	return Statistics{
		Loadavg: Loadavg{
			One:              1.2,
			Five:             2.34,
			Fifteen:          5.67,
			CurrentProcesses: 1,
			TotalProcesses:   876,
			LatestPID:        1234,
		},
		Interfaces: map[string]Interface{
			infName: Interface{
				Speed:    infSpeed,
				BytesOut: outBytes,
				BytesIn:  12234567,
			},
		},
		NotAvailable: randBool(),
	}

}

func TestAstatsPrecompute(t *testing.T) {
	dsNameFQDNs := getMockTODataDSNameDirectMatches()
	toData := getMockTOData(dsNameFQDNs)
	cacheName := "cache0"
	rawStats := getMockRawStats(cacheName, dsNameFQDNs)
	outBytes := uint64(987655443321)
	infSpeedMbps := int64(9876554433210)
	stats := getMockStatistics(infSpeedMbps, outBytes)

	prc := astatsPrecompute(cacheName, toData, stats, rawStats)

	if len(prc.Errors) != 0 {
		t.Fatalf("astatsPrecompute Errors expected 0, actual: %+v\n", prc.Errors)
	}
	if prc.OutBytes != outBytes {
		t.Fatalf("astatsPrecompute OutBytes expected 987655443321, actual: %+v\n", prc.OutBytes)
	}
	if prc.MaxKbps != infSpeedMbps*1000 {
		t.Fatalf("astatsPrecompute MaxKbps expected 9876554433210000, actual: %+v\n", prc.MaxKbps)
	}

	for dsName, dsFQDN := range dsNameFQDNs {
		dsStat, ok := prc.DeliveryServiceStats[string(dsName)]
		if !ok {
			t.Fatalf("astatsPrecompute DeliveryServiceStats expected %+v, actual: missing\n", dsName)
		}
		if statName := "plugin.remap_stats." + dsFQDN + ".in_bytes"; dsStat.InBytes != uint64(rawStats[statName].(float64)) {
			t.Fatalf("astatsPrecompute DeliveryServiceStats[%+v].InBytes expected %+v, actual: %+v\n", dsName, uint64(rawStats[statName].(float64)), dsStat.InBytes)
		}
		if statName := "plugin.remap_stats." + dsFQDN + ".out_bytes"; dsStat.OutBytes != uint64(rawStats[statName].(float64)) {
			t.Fatalf("astatsPrecompute DeliveryServiceStats[%+v].OutBytes expected %+v, actual: %+v\n", dsName, uint64(rawStats[statName].(float64)), dsStat.OutBytes)
		}
		if statName := "plugin.remap_stats." + dsFQDN + ".status_2xx"; dsStat.Status2xx != uint64(rawStats[statName].(float64)) {
			t.Fatalf("astatsPrecompute DeliveryServiceStats[%+v].Status2xx expected %+v, actual: %+v\n", dsName, uint64(rawStats[statName].(float64)), dsStat.Status2xx)
		}
		if statName := "plugin.remap_stats." + dsFQDN + ".status_3xx"; dsStat.Status3xx != uint64(rawStats[statName].(float64)) {
			t.Fatalf("astatsPrecompute DeliveryServiceStats[%+v].Status3xx expected %+v, actual: %+v\n", dsName, uint64(rawStats[statName].(float64)), dsStat.Status3xx)
		}
		if statName := "plugin.remap_stats." + dsFQDN + ".status_4xx"; dsStat.Status4xx != uint64(rawStats[statName].(float64)) {
			t.Fatalf("astatsPrecompute DeliveryServiceStats[%+v].Status4xx expected %+v, actual: %+v\n", dsName, uint64(rawStats[statName].(float64)), dsStat.Status4xx)
		}
		if statName := "plugin.remap_stats." + dsFQDN + ".status_5xx"; dsStat.Status5xx != uint64(rawStats[statName].(float64)) {
			t.Fatalf("astatsPrecompute DeliveryServiceStats[%+v].Status5xx expected %+v, actual: %+v\n", dsName, uint64(rawStats[statName].(float64)), dsStat.Status5xx)
		}
	}
}
