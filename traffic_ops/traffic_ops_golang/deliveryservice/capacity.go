package deliveryservice

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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/monitorhlp"
)

func GetCapacity(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	dsID := inf.IntParams["id"]

	userErr, sysErr, errCode = tenant.CheckID(inf.Tx.Tx, inf.User, dsID)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	ds, cdn, ok, err := dbhelpers.GetDSNameAndCDNFromID(inf.Tx.Tx, dsID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service name from ID: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
	}

	capacity, err := getCapacity(inf.Tx.Tx, ds, cdn)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service capacity: "+err.Error()))
		return
	}

	api.WriteResp(w, r, capacity)
}

type CapacityResp struct {
	AvailablePercent   float64 `json:"availablePercent"`
	UnavailablePercent float64 `json:"unavailablePercent"`
	UtilizedPercent    float64 `json:"utilizedPercent"`
	MaintenancePercent float64 `json:"maintenancePercent"`
}

type CapData struct {
	Available   float64
	Unavailable float64
	Maintenance float64
	Capacity    float64
}

func getCapacity(tx *sql.Tx, ds tc.DeliveryServiceName, cdn tc.CDNName) (CapacityResp, error) {
	monitors, err := monitorhlp.GetURLs(tx)
	if err != nil {
		return CapacityResp{}, errors.New("getting monitor URLs: " + err.Error())
	}
	client, err := monitorhlp.GetClient(tx)
	if err != nil {
		return CapacityResp{}, errors.New("getting monitor client: " + err.Error())
	}

	thresholds, err := getEdgeProfileHealthThresholdBandwidth(tx)
	if err != nil {
		return CapacityResp{}, errors.New("getting profile thresholds: " + err.Error())
	}

	monitorFQDN, ok := monitors[cdn]
	if !ok {
		return CapacityResp{}, nil // TODO emulates perl; change to error?
	}

	crStates, err := monitorhlp.GetCRStates(monitorFQDN, client)
	// TODO on err, try another online monitor
	if err != nil {
		return CapacityResp{}, errors.New("getting CRStates for delivery service '" + string(ds) + "' monitor '" + monitorFQDN + "': " + err.Error())
	}
	crConfig, err := monitorhlp.GetCRConfig(monitorFQDN, client)
	// TODO on err, try another online monitor
	if err != nil {
		return CapacityResp{}, errors.New("getting CRConfig for delivery service '" + string(ds) + "' monitor '" + monitorFQDN + "': " + err.Error())
	}
	statsoFetch := []string{tc.StatNameMaxKBPS, tc.StatNameKBPS}
	cacheStats, _, err := monitorhlp.GetCacheStats(monitorFQDN, client, statsoFetch)
	if err != nil {
		legacyCacheStats, _, err := monitorhlp.GetLegacyCacheStats(monitorFQDN, client, statsoFetch)
		if err != nil {
			return CapacityResp{}, errors.New("getting CacheStats for delivery service '" + string(ds) + "' monitor '" + monitorFQDN + "': " + err.Error())
		}
		cacheStats = monitorhlp.UpgradeLegacyStats(legacyCacheStats)
	}
	cap := addCapacity(CapData{}, ds, cacheStats, crStates, crConfig, thresholds)
	if cap.Capacity == 0 {
		if dsHasServer(ds, crConfig) {
			return CapacityResp{}, errors.New("Delivery service '" + string(ds) + "' has servers, but capacity was zero!'")
		}
		log.Warnf("Delivery service '" + string(ds) + "' has no servers. Returning 0 capacity'")
		return CapacityResp{}, nil // avoid divide-by-zero below.
	}

	return CapacityResp{
		UtilizedPercent:    (cap.Available * 100) / cap.Capacity,
		UnavailablePercent: (cap.Unavailable * 100) / cap.Capacity,
		MaintenancePercent: (cap.Maintenance * 100) / cap.Capacity,
		AvailablePercent:   ((cap.Capacity - cap.Unavailable - cap.Maintenance - cap.Available) * 100) / cap.Capacity,
	}, nil
}

// dsHasServer checks whether a given DS has servers.
func dsHasServer(ds tc.DeliveryServiceName, crConfig tc.CRConfig) bool {
	for _, server := range crConfig.ContentServers {
		if _, ok := server.DeliveryServices[string(ds)]; ok {
			return true
		}
	}
	return false

}

const StatNameKBPS = "kbps"
const StatNameMaxKBPS = "maxKbps"

func addCapacity(
	cap CapData,
	ds tc.DeliveryServiceName,
	cacheStats tc.Stats,
	crStates tc.CRStates,
	crConfig tc.CRConfig,
	thresholds map[string]float64,
) CapData {
	for cacheName, statsCache := range cacheStats.Caches {
		cache, ok := crConfig.ContentServers[string(cacheName)]
		if !ok {
			log.Warnln("Getting delivery service capacity: delivery service '" + string(ds) + "' cache '" + string(cacheName) + "' in CacheStats but not CRConfig, skipping")
			continue
		}

		if _, ok := cache.DeliveryServices[string(ds)]; !ok {
			continue
		}
		if cache.ServerType == nil || !strings.HasPrefix(string(*cache.ServerType), string(tc.CacheTypeEdge)) {
			continue
		}

		stat := statsCache.Stats
		if len(stat[tc.StatNameKBPS]) < 1 || len(stat[tc.StatNameMaxKBPS]) < 1 {
			log.Warnln("Getting delivery service capacity: delivery service '" + string(ds) + "' cache '" + string(cacheName) + "' CacheStats has no kbps or maxKbps, skipping")
			continue
		}

		kbps, err := statToFloat(stat[tc.StatNameKBPS][0].Val)
		if err != nil {
			log.Warnln("Getting delivery service capacity: delivery service '" + string(ds) + "' cache '" + string(cacheName) + "' CacheStats kbps is not a number, skipping")
			continue
		}
		maxKBPS, err := statToFloat(stat[tc.StatNameMaxKBPS][0].Val)
		if err != nil {
			log.Warnln("Getting delivery service capacity: delivery service '" + string(ds) + "' cache '" + string(cacheName) + "' CacheStats maxKps is not a number, skipping")
			continue
		}
		if cache.ServerStatus == nil {
			log.Warnln("Getting delivery service capacity: delivery service '" + string(ds) + "' cache '" + string(cacheName) + "' CRConfig Status is nil, skipping")
			continue
		}
		if cache.Profile == nil {
			log.Warnln("Getting delivery service capacity: delivery service '" + string(ds) + "' cache '" + string(cacheName) + "' CRConfig Profile is nil, skipping")
			continue
		}
		if tc.CacheStatus(*cache.ServerStatus) == tc.CacheStatusReported || tc.CacheStatus(*cache.ServerStatus) == tc.CacheStatusOnline {
			if crStates.Caches[tc.CacheName(cacheName)].IsAvailable {
				cap.Available += kbps
			} else {
				cap.Unavailable += kbps
			}
		} else if tc.CacheStatus(*cache.ServerStatus) == tc.CacheStatusAdminDown {
			cap.Maintenance += kbps
		} else {
			continue // don't add capacity for OFFLINE or other statuses
		}
		cap.Capacity += maxKBPS - thresholds[*cache.Profile]

	}
	return cap
}

// statToFloat converts a CacheStats stat interface{} to a float64
func statToFloat(s interface{}) (float64, error) {
	switch v := s.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		iv, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0.0, errors.New("stat is a string which is not a number: " + err.Error())
		}
		return iv, nil
	default:
		return 0.0, fmt.Errorf("unknown stat type: %T", s)
	}
}

func getEdgeProfileHealthThresholdBandwidth(tx *sql.Tx) (map[string]float64, error) {
	rows, err := tx.Query(`
SELECT pr.name as profile, pa.name, pa.config_file, pa.value
FROM parameter as pa
JOIN profile_parameter as pp ON pp.parameter = pa.id
JOIN profile as pr ON pp.profile = pr.id
JOIN server as s ON s.profile = pr.id
JOIN cdn as c ON c.id = s.cdn_id
JOIN type as t ON s.type = t.id
WHERE t.name LIKE 'EDGE%'
AND pa.config_file = 'rascal-config.txt'
AND pa.name = 'health.threshold.availableBandwidthInKbps'
`)
	if err != nil {
		return nil, errors.New("querying thresholds: " + err.Error())
	}
	defer rows.Close()
	profileThresholds := map[string]float64{}
	for rows.Next() {
		profile := ""
		threshStr := ""
		if err := rows.Scan(&profile, &threshStr); err != nil {
			return nil, errors.New("scanning thresholds: " + err.Error())
		}
		threshStr = strings.TrimPrefix(threshStr, ">")
		thresh, err := strconv.ParseFloat(threshStr, 64)
		if err != nil {
			return nil, errors.New("profile '" + profile + "' health.threshold.availableBandwidthInKbps is not a number")
		}
		profileThresholds[profile] = thresh
	}
	return profileThresholds, nil
}
