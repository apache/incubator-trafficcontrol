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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	influx "github.com/influxdata/influxdb/client/v2"
)

const (
	dsTenantIDFromXMLIDQuery = `
		SELECT tenant_id
		FROM deliveryservice
		WHERE xml_id = $1`

	xmlidFromIDQuery = `
		SELECT xml_id
		FROM deliveryservice
		WHERE id = $1`

	// TODO: Pretty sure all of this could actually be calculated using the fetched data (assuming an
	// interval is given). Check to see if that's faster than doing another synchronous HTTP request.
	dsSummaryQuery = `
		SELECT mean(value) AS "average",
		       percentile(value, 5) AS "fifthPercentile",
		       percentile(value, 95) AS "ninetyFifthPercentile",
		       percentile(value, 98) AS "ninetyEighthPercentile",
		       min(value) AS "min",
		       max(value) AS "max",
		       count(value) AS "count"
		FROM "%s"."monthly"."%s.ds.1min"
		WHERE time >= $start
		AND time <= $end
		AND cachegroup = 'total'
		AND deliveryservice = $xmlid`

	dsSeriesQuery = `
		SELECT mean(value)
		FROM "%s"."monthly"."%s.ds.1min"
		WHERE cachegroup = 'total'
		AND deliveryservice = $xmlid
		AND time >= $start
		AND time <= $end
		GROUP BY time(%s, %s), cachegroup%s`
)

func dsConfigFromRequest(r *http.Request, i *api.Info) (tc.TrafficDSStatsConfig, int, error) {
	c := tc.TrafficDSStatsConfig{}
	statsConfig, rc, e := tsConfigFromRequest(r, i)
	if e != nil {
		return c, rc, e
	}
	c.TrafficStatsConfig = statsConfig
	c.MetricType = i.Params["metricType"]
	if _, found := findMetric(i.Config.ConfigTrafficOpsGolang.SupportedDSMetrics, c.MetricType); !found {
		e = fmt.Errorf("Metric is not supported: %s", c.MetricType)
		return c, http.StatusBadRequest, e
	}

	var ok bool
	if c.DeliveryService, ok = i.Params["deliveryServiceName"]; !ok {
		if c.DeliveryService, ok = i.Params["deliveryService"]; !ok {
			e = errors.New("You must specify deliveryService or deliveryServiceName!")
			return c, http.StatusBadRequest, e
		}

		if dsID, err := strconv.ParseUint(c.DeliveryService, 10, 64); err == nil {
			// sql.ErrNoRows does not *necessarily* mean the DS doesn't exist - an XMLID can simply
			// be numeric, and so it was wrong to treat it as an ID in the first place.
			xmlid := c.DeliveryService
			var exists bool
			if exists, c.DeliveryService, err = getXMLIDFromID(dsID, i.Tx.Tx); err != nil {
				log.Errorf("Converting DSID to XMLID: %v", err)
				e = errors.New("Internal Server Error")
				return c, http.StatusInternalServerError, e
			} else if !exists {
				c.DeliveryService = xmlid
			}
		}
	}

	return c, http.StatusOK, nil
}

// GetDSStats handler for getting deliveryservice stats
func GetDSStats(w http.ResponseWriter, r *http.Request) {
	// Perl didn't require "interval", but it would only return summary data if it was not given
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"metricType", "startDate", "endDate"}, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var c tc.TrafficDSStatsConfig
	if c, errCode, userErr = dsConfigFromRequest(r, inf); userErr != nil {
		sysErr = fmt.Errorf("Unable to process deliveryservice_stats request: %v", userErr)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	client, err := inf.CreateInfluxClient()
	if err != nil {
		errCode = http.StatusInternalServerError
		sysErr = err
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	} else if client == nil {
		sysErr = errors.New("Traffic Stats is not configured, but DS stats were requested")
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}
	defer (*client).Close()

	exists, dsTenant, err := dsTenantIDFromXMLID(c.DeliveryService, tx)
	if err != nil {
		sysErr = err
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	} else if !exists {
		userErr = fmt.Errorf("No such Delivery Service: %s", c.DeliveryService)
		errCode = http.StatusNotFound
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	authorized, err := tenant.IsResourceAuthorizedToUserTx(int(dsTenant), inf.User, tx)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	} else if !authorized {
		// If the Tenant is not authorized to use the resource, then we DON'T tell them that.
		// Instead, we don't disclose that such a Delivery Service exists at all - in keeping with
		// the behavior of /deliveryservices
		// This is different from what Perl used to do, but then again Perl didn't check tenancy at
		// all.
		userErr = fmt.Errorf("No such Delivery Service: %s", c.DeliveryService)
		sysErr = fmt.Errorf("GetDSStats: unauthorized Tenant (#%d) access", inf.User.TenantID)
		errCode = http.StatusNotFound
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if inf.Version.Major > 1 {
		handleRequest(w, r, client, c, inf)
	} else {
		handleLegacyRequest(w, r, client, c, inf)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, client *influx.Client, cfg tc.TrafficDSStatsConfig, inf *api.Info) {
	// TODO: as above, this could be done on TO itself, thus sending only one synchronous request
	// per hit on this endpoint, rather than the current two. Not sure if that's worth it for large
	// data sets, though.
	var resp tc.TrafficDSStatsResponse
	if !cfg.ExcludeSummary {
		summary, kBs, txns, err := getDSSummary(client, &cfg, inf.Config.ConfigInflux.DSDBName)

		if err != nil {
			sysErr := fmt.Errorf("Getting summary response from Influx: %v", err)
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, sysErr)
			return
		}

		// match Perl implementation and set summary to zero values if no data
		if summary != nil {
			resp.Summary = &tc.TrafficDSStatsSummary{
				TrafficStatsSummary: *summary,
				TotalKiloBytes:      kBs,
				TotalTransactions:   txns,
			}
		} else {
			resp.Summary = &tc.TrafficDSStatsSummary{}
		}

	}

	if !cfg.ExcludeSeries {
		series, err := getDSSeries(client, &cfg, inf.Config.ConfigInflux.DSDBName)

		if err != nil {
			sysErr := fmt.Errorf("Getting summary response from Influx: %v", err)
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, sysErr)
			return
		}

		// match Perl implementation and omit series if no data
		if series != nil {
			if !cfg.Unix {
				series.FormatTimestamps()
			}

			resp.Series = series
		}
	}

	var respObj struct {
		Response interface{} `json:"response"`
	}
	respObj.Response = resp

	respBts, err := json.Marshal(respObj)
	if err != nil {
		sysErr := fmt.Errorf("Marshalling response: %v", err)
		errCode := http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	if cfg.Unix {
		w.Header().Set(rfc.ContentType, jsonWithUnixTimestamps.String())
	} else {
		w.Header().Set(rfc.ContentType, jsonWithRFCTimestamps.String())
	}
	w.Header().Set(http.CanonicalHeaderKey("vary"), http.CanonicalHeaderKey("Accept"))
	w.Write(append(respBts, '\n'))
}

func handleLegacyRequest(w http.ResponseWriter, r *http.Request, client *influx.Client, cfg tc.TrafficDSStatsConfig, inf *api.Info) {
	// TODO: as above, this could be done on TO itself, thus sending only one synchronous request
	// per hit on this endpoint, rather than the current two. Not sure if that's worth it for large
	// data sets, though.
	var resp tc.TrafficDSStatsResponseV1
	if !cfg.ExcludeSummary {
		summary, kBs, txns, err := getDSSummary(client, &cfg, inf.Config.ConfigInflux.DSDBName)

		if err != nil {
			sysErr := fmt.Errorf("Getting summary response from Influx: %v", err)
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, sysErr)
			return
		}

		// match Perl implementation and set summary to zero values if no data
		if summary != nil {
			resp.Summary = &tc.LegacyTrafficDSStatsSummary{
				TrafficStatsSummary: *summary,
				TotalBytes:          kBs,
				TotalTransactions:   txns,
			}
		} else {
			resp.Summary = &tc.LegacyTrafficDSStatsSummary{}
		}

	}

	if !cfg.ExcludeSeries {
		series, err := getDSSeries(client, &cfg, inf.Config.ConfigInflux.DSDBName)

		if err != nil {
			sysErr := fmt.Errorf("Getting summary response from Influx: %v", err)
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, sysErr)
			return
		}

		// match Perl implementation and omit series if no data
		if series != nil {
			if !cfg.Unix {
				series.FormatTimestamps()
			}

			resp.Series = series
		}
	}

	var respObj struct {
		Response interface{} `json:"response"`
	}
	respObj.Response = resp

	respBts, err := json.Marshal(respObj)
	if err != nil {
		sysErr := fmt.Errorf("Marshalling response: %v", err)
		errCode := http.StatusInternalServerError
		api.HandleErr(w, r, inf.Tx.Tx, errCode, nil, sysErr)
		return
	}

	if cfg.Unix {
		w.Header().Set(rfc.ContentType, jsonWithUnixTimestamps.String())
	} else {
		w.Header().Set(rfc.ContentType, jsonWithRFCTimestamps.String())
	}
	w.Header().Set(http.CanonicalHeaderKey("vary"), http.CanonicalHeaderKey("Accept"))
	w.Write(append(respBts, '\n'))
}

func getDSSummary(client *influx.Client, conf *tc.TrafficDSStatsConfig, db string) (*tc.TrafficStatsSummary, *float64, *float64, error) {
	qStr := fmt.Sprintf(dsSummaryQuery, db, conf.MetricType)
	q := influx.NewQueryWithParameters(qStr,
		db,
		"rfc3339", //this doesn't actually seem to have any effect...
		map[string]interface{}{
			"xmlid":    conf.DeliveryService,
			"start":    conf.Start,
			"end":      conf.End,
			"interval": string(conf.Interval),
		})
	ts, err := getSummary(db, q, client)
	if err != nil || ts == nil {
		return nil, nil, nil, err
	}

	var totalKB *float64
	var totalTXN *float64
	value := float64(ts.Count*60) * ts.Average
	if strings.HasPrefix(conf.MetricType, "kbps") {
		// TotalBytes is actually in units of kB....
		value /= 8
		totalKB = &value
	} else {
		totalTXN = &value
	}

	return ts, totalKB, totalTXN, nil
}

func dsTenantIDFromXMLID(xmlid string, tx *sql.Tx) (bool, uint, error) {
	row := tx.QueryRow(dsTenantIDFromXMLIDQuery, xmlid)
	var tid uint
	err := row.Scan(&tid)
	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	return true, tid, err
}

func getXMLIDFromID(id uint64, tx *sql.Tx) (bool, string, error) {
	row := tx.QueryRow(xmlidFromIDQuery, id)
	var xmlid string
	err := row.Scan(&xmlid)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	return true, xmlid, err
}

func getDSSeries(client *influx.Client, conf *tc.TrafficDSStatsConfig, db string) (*tc.TrafficStatsSeries, error) {
	extraClauses := buildExtraClauses(&conf.TrafficStatsConfig)
	qStr := fmt.Sprintf(dsSeriesQuery, db, conf.MetricType, conf.Interval, conf.TrafficStatsConfig.OffsetString(), extraClauses)
	q := influx.NewQueryWithParameters(qStr,
		db,
		"rfc3339", // this doesn't seem to do anything...
		map[string]interface{}{
			"xmlid": conf.DeliveryService,
			"start": conf.Start,
			"end":   conf.End,
		})
	return getSeries(db, q, client)
}

func findMetric(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}
