package trafficstats

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
	"net/http"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/lib/pq"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

const (
	cdnStatsQuery = `
SELECT last(value) FROM "%s"."monthly"."%s"
	WHERE cdn = $cdn`
	bwMetricName   = "bandwidth.cdn.1min"
	connMetricName = "connections.cdn.1min"
	kbpsMetricName = "maxkbps.cdn.1min"
)

// GetCurrentStats handler for getting current stats for CDNs
func GetCurrentStats(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	tx := inf.Tx.Tx
	var sysErr error
	client, err := inf.CreateInfluxClient()
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	} else if client == nil {
		sysErr = errors.New("Traffic Stats is not configured and 'current_stats' was requested")
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}
	defer (*client).Close()

	currentStats := []interface{}{}

	// Get CDN names
	cdns := []string{}
	if err := tx.QueryRow(`SELECT ARRAY(SELECT name FROM cdn)`).Scan(pq.Array(&cdns)); err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("querying cdn names"))
		return
	}
	totalStats := tc.TrafficStatsTotalStats{
		CDN: "total",
	}

	for _, cdn := range cdns {
		cdnStats := tc.TrafficStatsCDNStats{
			CDN: cdn,
		}
		bw, err := getCDNStat(client, cdn, bwMetricName, inf.Config.ConfigInflux.CacheDBName)
		if err != nil {
			sysErr = fmt.Errorf("getting bandwidth from cdn %v: %v", cdn, err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		if bw != nil {
			if totalStats.Bandwidth == nil {
				totalStats.Bandwidth = util.FloatPtr(0.0)
			}
			*totalStats.Bandwidth += *bw
		}

		if bw != nil {
			cdnStats.Bandwidth = util.FloatPtr(*bw / 1000000)
		}

		con, err := getCDNStat(client, cdn, connMetricName, inf.Config.ConfigInflux.CacheDBName)
		if err != nil {
			sysErr = fmt.Errorf("getting connections from cdn %v: %v", cdn, err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		if con != nil {
			if totalStats.Connnections == nil {
				totalStats.Connnections = util.FloatPtr(0.0)
			}
			*totalStats.Connnections += *con
		}
		cdnStats.Connnections = con

		cap, err := getCDNStat(client, cdn, kbpsMetricName, inf.Config.ConfigInflux.CacheDBName)
		if err != nil {
			sysErr = fmt.Errorf("getting maxkbps from cdn %v: %v", cdn, err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		if cap != nil {
			// Perl implementation hardcoded capacity as 85 percent of Gbps
			cdnStats.Capacity = util.FloatPtr(*cap / 1000000 * .85)
		}

		currentStats = append(currentStats, cdnStats)
	}

	if totalStats.Bandwidth != nil {
		*totalStats.Bandwidth /= 1000000
	}
	currentStats = append(currentStats, totalStats)
	resp := struct {
		CurrentStats []interface{} `json:"currentStats"`
	}{
		CurrentStats: currentStats,
	}

	api.WriteResp(w, r, resp)
}

func getCDNStat(client *influx.Client, cdnName, metricName, db string) (*float64, error) {
	qStr := fmt.Sprintf(cdnStatsQuery, db, metricName)
	q := influx.NewQueryWithParameters(qStr,
		db,
		"rfc3339",
		map[string]interface{}{
			"cdn": cdnName,
		})
	series, err := getSeries(db, q, client)
	if err != nil {
		return nil, err
	}
	if series == nil {
		return nil, nil
	}
	if len(series.Values) == 0 {
		return nil, fmt.Errorf("influxdb query for metrtic %v returned series with no values", metricName)
	}
	vals := series.Values[0]
	mappedValues := map[string]interface{}{}
	for i, v := range vals {
		mappedValues[series.Columns[i]] = v
	}
	val, err := extractFloat64("last", mappedValues)
	if err != nil {
		return nil, err
	}
	return &val, nil
}
