package cdn

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
	"net/http"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/monitorhlp"
)

func GetHealth(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	health, err := getHealth(inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cdn health: "+err.Error()))
		return
	}

	api.WriteResp(w, r, health)
}

func GetNameHealth(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	health, err := getNameHealth(inf.Tx.Tx, tc.CDNName(inf.Params["name"]))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cdn name health: "+err.Error()))
		return
	}

	api.WriteResp(w, r, health)
}

func getHealth(tx *sql.Tx) (tc.HealthData, error) {
	monitors, err := monitorhlp.GetURLs(tx)
	if err != nil {
		return tc.HealthData{}, errors.New("getting monitors: " + err.Error())
	}
	return getMonitorsHealth(tx, monitors)
}

func getNameHealth(tx *sql.Tx, name tc.CDNName) (tc.HealthData, error) {
	monitors, err := monitorhlp.GetURLs(tx)
	if err != nil {
		return tc.HealthData{}, errors.New("getting monitors: " + err.Error())
	}
	monitor, ok := monitors[name]
	monitors = nil
	if ok {
		monitors = map[tc.CDNName]string{name: monitor}
	}
	return getMonitorsHealth(tx, monitors)
}

func getMonitorsHealth(tx *sql.Tx, monitors map[tc.CDNName]string) (tc.HealthData, error) {
	client, err := monitorhlp.GetClient(tx)
	if err != nil {
		return tc.HealthData{}, errors.New("getting monitor client: " + err.Error())
	}

	totalOnline := uint64(0)
	totalOffline := uint64(0)
	cgData := map[tc.CacheGroupName]tc.HealthDataCacheGroup{}
	for cdn, monitorFQDN := range monitors {
		crStates, err := monitorhlp.GetCRStates(monitorFQDN, client)
		// TODO on err, try another online monitor
		if err != nil {
			return tc.HealthData{}, errors.New("getting CRStates for CDN '" + string(cdn) + "' monitor '" + monitorFQDN + "': " + err.Error())
		}
		crConfig, err := monitorhlp.GetCRConfig(monitorFQDN, client)
		// TODO on err, try another online monitor
		if err != nil {
			return tc.HealthData{}, errors.New("getting CRConfig for CDN '" + string(cdn) + "' monitor '" + monitorFQDN + "': " + err.Error())
		}
		cgData, totalOnline, totalOffline = addHealth(cgData, totalOnline, totalOffline, crStates, crConfig)
	}

	healthData := tc.HealthData{TotalOffline: totalOffline, TotalOnline: totalOnline}
	for _, health := range cgData {
		healthData.CacheGroups = append(healthData.CacheGroups, health)
	}
	return healthData, nil
}

// addHealth adds the given cache states to the given data and totals, and returns the new data and totals
func addHealth(data map[tc.CacheGroupName]tc.HealthDataCacheGroup, totalOnline uint64, totalOffline uint64, crStates tc.CRStates, crConfig tc.CRConfig) (map[tc.CacheGroupName]tc.HealthDataCacheGroup, uint64, uint64) {
	for cacheName, avail := range crStates.Caches {
		cache, ok := crConfig.ContentServers[string(cacheName)]
		if !ok {
			continue
		}
		if cache.ServerStatus == nil || *cache.ServerStatus != tc.CRConfigServerStatus(tc.CacheStatusReported) {
			continue
		}
		if cache.ServerType == nil || !strings.HasPrefix(string(*cache.ServerType), string(tc.CacheTypeEdge)) {
			continue
		}
		if cache.CacheGroup == nil {
			continue // TODO warn?
		}

		cgHealth := data[tc.CacheGroupName(*cache.CacheGroup)]
		cgHealth.Name = tc.CacheGroupName(*cache.CacheGroup)
		if avail.IsAvailable {
			cgHealth.Online++
			totalOnline++
		} else {
			cgHealth.Offline++
			totalOffline++
		}
		data[tc.CacheGroupName(*cache.CacheGroup)] = cgHealth
	}
	return data, totalOnline, totalOffline
}
