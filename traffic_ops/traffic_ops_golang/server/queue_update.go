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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

// QueueUpdateHandler implements an http handler that updates a server's
// upd_pending value.
func QueueUpdateHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var reqObj tc.ServerQueueUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObj); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, fmt.Errorf("malformed JSON: %v", err), nil)
		return
	}

	if reqObj.Action != "queue" && reqObj.Action != "dequeue" {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("action must be 'queue' or 'dequeue'"), nil)
		return
	}

	serverID := int64(inf.IntParams["id"])
	queue := reqObj.Action == "queue"
	ok, err := queueUpdate(inf.Tx.Tx, serverID, queue)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("queueing updates: %v", err))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, fmt.Errorf("no server with id '%v' found", serverID), nil)
		return
	}

	err = api.CreateChangeLogBuildMsg(
		api.ApiChange,
		api.Updated,
		inf.User,
		inf.Tx.Tx,
		"server",
		fmt.Sprint(serverID),
		map[string]interface{}{"id": serverID},
	)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("writing changelog: %v", err))
		return
	}

	api.WriteResp(w, r, tc.ServerQueueUpdate{
		ServerID: util.JSONIntStr(serverID),
		Action:   reqObj.Action,
	})
}

// queueUpdate sets the upd_pending column of a server to the value of queue. It
// returns true if the identified server exists and was updated and false if no
// server was updated either because it doesn't exist or there was an error.
func queueUpdate(tx *sql.Tx, serverID int64, queue bool) (bool, error) {
	const query = `UPDATE server SET upd_pending = $1 WHERE id = $2`

	if result, err := tx.Exec(query, queue, serverID); err != nil {
		return false, fmt.Errorf("updating server table: %v", err)
	} else if rc, err := result.RowsAffected(); err != nil {
		return false, fmt.Errorf("checking rows updated: %v", err)
	} else {
		return rc == 1, nil
	}
}
