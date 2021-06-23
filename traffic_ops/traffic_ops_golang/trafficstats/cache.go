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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	influx "github.com/influxdata/influxdb/client/v2"
)

var (
	cMetricTypes = map[string]interface{}{
		"maxkbps":     struct{}{},
		"connections": struct{}{},
		"bandwidth":   struct{}{},
	}
)

const (
	cSummaryQuery = `
SELECT mean(value) AS "average",
	percentile(value, 5) AS "fifthPercentile",
	percentile(value, 95) AS "ninetyFifthPercentile",
	percentile(value, 98) AS "ninetyEighthPercentile",
	min(value) AS "min",
	max(value) AS "max",
	count(value) AS "count"
FROM "%s"."monthly"."%s.cdn.1min"
WHERE cdn = $cdn_name
AND time < $end
AND time > $start`

	cSeriesQuery = `
SELECT sum(value)/count(value)
FROM "%s"."monthly"."%s.cdn.1min"
WHERE cdn = $cdn_name
AND time > $start
AND time < $end
GROUP BY time(%s, %s), cdn%s`
)

func cacheConfigFromRequest(r *http.Request, i *api.Info) (tc.TrafficCacheStatsConfig, int, error) {
	c := tc.TrafficCacheStatsConfig{}
	statsConfig, rc, e := tsConfigFromRequest(r, i)
	if e != nil {
		return c, rc, e
	}
	c.TrafficStatsConfig = statsConfig
	c.MetricType = i.Params["metricType"]
	if _, ok := cMetricTypes[c.MetricType]; !ok {
		e = fmt.Errorf("Unknown metric type: %s", c.MetricType)
		return c, http.StatusBadRequest, e
	}

	var ok bool
	if c.CDN, ok = i.Params["cdnName"]; !ok {
		e = errors.New("you must specify cdnName")
		return c, http.StatusBadRequest, e
	}

	return c, http.StatusOK, nil
}

// GetCacheStats handler for getting cache stats
func GetCacheStats(w http.ResponseWriter, r *http.Request) {
	// Perl didn't require "interval", but it would only return summary data if it was not given
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"metricType", "startDate", "endDate", "cdnName"}, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var c tc.TrafficCacheStatsConfig
	if c, errCode, userErr = cacheConfigFromRequest(r, inf); userErr != nil {
		sysErr = fmt.Errorf("Unable to process cache_stats request: %v", userErr)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	exists, err := dbhelpers.CDNExists(c.CDN, tx)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	} else if !exists {
		userErr = fmt.Errorf("no such CDN: %s", c.CDN)
		api.HandleErr(w, r, tx, http.StatusNotFound, userErr, nil)
		return
	}

	client, err := inf.CreateInfluxClient()
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	} else if client == nil {
		sysErr = errors.New("Traffic Stats is not configured, but Cache stats were requested")
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}
	defer (*client).Close()

	resp := struct {
		Response tc.TrafficStatsResponse `json:"response"`
	}{
		Response: tc.TrafficStatsResponse{
			Series:  nil,
			Summary: nil,
		},
	}

	if !c.ExcludeSummary {
		summary, err := getCacheSummary(client, &c, inf.Config.ConfigInflux.CacheDBName)

		if err != nil {
			sysErr = fmt.Errorf("Getting summary response from Influx: %v", err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}

		// match Perl implementation and set summary to zero values if no data
		if summary != nil {
			resp.Response.Summary = summary
		} else {
			resp.Response.Summary = &tc.TrafficStatsSummary{}
		}
	}

	if !c.ExcludeSeries {
		series, err := getCacheSeries(client, &c, inf.Config.ConfigInflux.CacheDBName)

		if err != nil {
			sysErr = fmt.Errorf("Getting summary response from Influx: %v", err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}

		// match Perl implementation and omit series if no data
		if series != nil {
			if !c.Unix {
				series.FormatTimestamps()
			}

			resp.Response.Series = series
		}
	}

	respBts, err := json.Marshal(resp)
	if err != nil {
		sysErr = fmt.Errorf("Marshalling response: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	if c.Unix {
		w.Header().Set(rfc.ContentType, jsonWithUnixTimestamps.String())
	} else {
		w.Header().Set(rfc.ContentType, jsonWithRFCTimestamps.String())
	}
	w.Header().Set(http.CanonicalHeaderKey("vary"), http.CanonicalHeaderKey("Accept"))
	w.Write(append(respBts, '\n'))
}

func getCacheSummary(client *influx.Client, conf *tc.TrafficCacheStatsConfig, db string) (*tc.TrafficStatsSummary, error) {
	qStr := fmt.Sprintf(cSummaryQuery, db, conf.MetricType)
	q := influx.NewQueryWithParameters(qStr,
		db,
		"rfc3339",
		map[string]interface{}{
			"cdn_name": conf.CDN,
			"start":    conf.Start,
			"end":      conf.End,
		})
	return getSummary(db, q, client)
}

func getCacheSeries(client *influx.Client, conf *tc.TrafficCacheStatsConfig, db string) (*tc.TrafficStatsSeries, error) {
	extraClauses := buildExtraClauses(&conf.TrafficStatsConfig)
	qStr := fmt.Sprintf(cSeriesQuery, db, conf.MetricType, conf.Interval, conf.TrafficStatsConfig.OffsetString(), extraClauses)
	q := influx.NewQueryWithParameters(qStr,
		db,
		"rfc3339",
		map[string]interface{}{
			"cdn_name": conf.CDN,
			"start":    conf.Start,
			"end":      conf.End,
		})

	return getSeries(db, q, client)
}
