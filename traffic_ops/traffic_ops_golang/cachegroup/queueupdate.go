package cachegroup

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
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

func QueueUpdates(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	reqObj := tc.CachegroupQueueUpdatesRequest{}
	if err := json.NewDecoder(r.Body).Decode(&reqObj); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	if reqObj.Action != "queue" && reqObj.Action != "dequeue" {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("action must be 'queue' or 'dequeue'"), nil)
		return
	}
	if reqObj.CDNID == nil && (reqObj.CDN == nil || *reqObj.CDN == "") {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("cdn is a required field"), nil)
		return
	}
	if reqObj.CDNID != nil && (reqObj.CDN == nil || *reqObj.CDN == "") {
		cdn, ok, err := dbhelpers.GetCDNNameFromID(inf.Tx.Tx, int64(*reqObj.CDNID))
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting CDN name from ID '"+strconv.Itoa(int(*reqObj.CDNID))+"': "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("cdn "+strconv.Itoa(int(*reqObj.CDNID))+" does not exist"), nil)
			return
		}
		reqObj.CDN = &cdn
	}
	cgID := inf.IntParams["id"]
	cgName, ok, err := dbhelpers.GetCacheGroupNameFromID(inf.Tx.Tx, cgID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cachegroup name from ID '"+inf.Params["id"]+"': "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}
	queue := reqObj.Action == "queue"
	updatedCaches, err := queueUpdates(inf.Tx.Tx, cgID, *reqObj.CDN, queue)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("queueing updates: "+err.Error()))
		return
	}

	api.WriteResp(w, r, QueueUpdatesResp{
		CacheGroupName: cgName,
		Action:         reqObj.Action,
		ServerNames:    updatedCaches,
		CDN:            *reqObj.CDN,
		CacheGroupID:   cgID,
	})
	api.CreateChangeLogRawTx(api.ApiChange, "CACHEGROUP: "+string(cgName)+", ID: "+strconv.Itoa(cgID)+", ACTION: "+strings.Title(reqObj.Action)+"d CacheGroup server updates to the "+string(*reqObj.CDN)+" CDN", inf.User, inf.Tx.Tx)
}

type QueueUpdatesResp struct {
	CacheGroupName tc.CacheGroupName `json:"cachegroupName"`
	Action         string            `json:"action"`
	ServerNames    []tc.CacheName    `json:"serverNames"`
	CDN            tc.CDNName        `json:"cdn"`
	CacheGroupID   int               `json:"cachegroupID"`
}

func queueUpdates(tx *sql.Tx, cgID int, cdn tc.CDNName, queue bool) ([]tc.CacheName, error) {
	q := `
UPDATE server SET upd_pending = $1
WHERE server.cachegroup = $2
AND server.cdn_id = (select id from cdn where name = $3)
RETURNING server.host_name
`
	rows, err := tx.Query(q, queue, cgID, cdn)
	if err != nil {
		return nil, errors.New("querying queue updates: " + err.Error())
	}
	defer rows.Close()
	names := []tc.CacheName{}
	for rows.Next() {
		name := ""
		if err := rows.Scan(&name); err != nil {
			return nil, errors.New("scanning queue updates: " + err.Error())
		}
		names = append(names, tc.CacheName(name))
	}
	return names, nil
}
