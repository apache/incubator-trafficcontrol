package server

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
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

func UpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"id"}, []string{"id"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()
	reqObj := tc.ServerPutStatus{}
	if err := json.NewDecoder(r.Body).Decode(&reqObj); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}

	serverInfo, exists, err := dbhelpers.GetServerInfo(inf.IntParams["id"], inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !exists {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, fmt.Errorf("server ID %d not found", inf.IntParams["id"]), nil)
		return
	}

	status := tc.StatusNullable{}
	statusExists := false
	if reqObj.Status.Name != nil {
		status, statusExists, err = dbhelpers.GetStatusByName(*reqObj.Status.Name, inf.Tx.Tx)
	} else if reqObj.Status.ID != nil {
		status, statusExists, err = dbhelpers.GetStatusByID(*reqObj.Status.ID, inf.Tx.Tx)
	} else {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("status is required"), nil)
		return
	}
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !statusExists {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("invalid status (does not exist)"), nil)
		return
	}

	if *status.Name == tc.CacheStatusAdminDown.String() || *status.Name == tc.CacheStatusOffline.String() {
		if reqObj.OfflineReason == nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("offlineReason is required for "+tc.CacheStatusAdminDown.String()+" or "+tc.CacheStatusOffline.String()+" status"), nil)
			return
		}
		*reqObj.OfflineReason = inf.User.UserName + ": " + *reqObj.OfflineReason
	} else {
		reqObj.OfflineReason = nil
	}
	if err := updateServerStatusAndOfflineReason(inf.IntParams["id"], *status.ID, reqObj.OfflineReason, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	offlineReason := ""
	if reqObj.OfflineReason != nil {
		offlineReason = *reqObj.OfflineReason
	}
	msg := "Updated status [ " + *status.Name + " ] for " + serverInfo.HostName + "." + serverInfo.DomainName + " [ " + offlineReason + " ]"

	// queue updates on child servers if server is ^EDGE or ^MID
	if strings.HasPrefix(serverInfo.Type, tc.CacheTypeEdge.String()) || strings.HasPrefix(serverInfo.Type, tc.CacheTypeMid.String()) {
		if err := queueUpdatesOnChildCaches(inf.Tx.Tx, serverInfo.CDNID, serverInfo.CachegroupID); err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
			return
		}
		msg += " and queued updates on all child caches"
	}
	api.CreateChangeLogRawTx(api.ApiChange, msg, inf.User, inf.Tx.Tx)
	api.WriteRespAlert(w, r, tc.SuccessLevel, msg)
}

// queueUpdatesOnChildCaches queues updates on child caches of the given cdnID and parentCachegroupID and returns an error (if one occurs).
func queueUpdatesOnChildCaches(tx *sql.Tx, cdnID, parentCachegroupID int) error {
	q := `
/* topology_descendants finds the descendant topology nodes of the topology node
 * for the cachegroup containing server $2.
 */
WITH RECURSIVE topology_descendants AS (
/* This is the base case of the recursive CTE, the topology node for the
 * cachegroup containing cachegroup $2.
 */
	SELECT tcp.parent child, NULL cachegroup
	FROM cachegroup c
	JOIN topology_cachegroup tc ON c."name" = tc.cachegroup
	JOIN topology_cachegroup_parents tcp ON tc.id = tcp.parent
	WHERE c.id = $2
UNION ALL
/* Find all direct topology child nodes tc of a given topology descendant td. */
	SELECT tcp.child, tc.cachegroup
	FROM topology_descendants td, topology_cachegroup_parents tcp
	JOIN topology_cachegroup tc ON tcp.child = tc.id
	WHERE td.child = tcp.parent
/* server_topology_descendants is the set of every server whose cachegroup is a
 * descendant topology node found by topology_descendants.
 */
), server_topology_descendants AS (
SELECT c.id
FROM cachegroup c
JOIN topology_descendants td ON c."name" = td.cachegroup
)
UPDATE server
SET upd_pending = TRUE
WHERE (server.cdn_id = $1
	   AND server.cachegroup IN (
			SELECT id
			FROM cachegroup
			WHERE parent_cachegroup_id = $2
				OR secondary_parent_cachegroup_id = $2
			))
		OR server.cachegroup IN (SELECT stc.id FROM server_topology_descendants stc)
`
	if _, err := tx.Exec(q, cdnID, parentCachegroupID); err != nil {
		return errors.New("queueing updates on child caches: " + err.Error())
	}
	return nil
}

// checkExistingStatusInfo returns the existing status and status_last_updated values for the server in question
func checkExistingStatusInfo(serverID int, tx *sql.Tx) (int, time.Time) {
	status := 0
	var status_last_updated time.Time
	q := `SELECT status,
status_last_updated
FROM server
WHERE id = $1`
	response, err := tx.Query(q, serverID)
	if err != nil {
		log.Errorf("couldn't get status/ status_last_updated for server with id %v", serverID)
		return status, status_last_updated
	}
	defer response.Close()
	for response.Next() {
		if err := response.Scan(&status, &status_last_updated); err != nil {
			log.Errorf("couldn't get status/ status_last_updated of server with id %v, err: %v", serverID, err.Error())
		}
	}
	return status, status_last_updated
}

// updateServerStatusAndOfflineReason updates a server's status and offline_reason and returns an error (if one occurs).
func updateServerStatusAndOfflineReason(serverID, statusID int, offlineReason *string, tx *sql.Tx) error {
	existingStatus, existingStatusUpdatedTime := checkExistingStatusInfo(serverID, tx)
	newStatusUpdatedTime := time.Now()
	// Set the status_last_updated time to the current time ONLY IF the new status is different from the old one
	if existingStatus == statusID {
		newStatusUpdatedTime = existingStatusUpdatedTime
	}
	q := `
UPDATE server
SET    status = $1,
       offline_reason = $2,
       status_last_updated = $3
WHERE  id = $4
`
	if _, err := tx.Exec(q, statusID, offlineReason, &newStatusUpdatedTime, serverID); err != nil {
		return errors.New("updating server status and offline_reason: " + err.Error())
	}
	return nil
}
