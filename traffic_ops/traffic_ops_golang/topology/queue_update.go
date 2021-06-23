package topology

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

	validation "github.com/go-ozzo/ozzo-validation"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

func Validate(reqObj tc.TopologiesQueueUpdateRequest, topologyName tc.TopologyName, tx *sql.Tx) error {
	errorMap := validation.Errors{}

	if reqObj.Action != "queue" && reqObj.Action != "dequeue" {
		errorMap["action"] = errors.New("action must be 'queue' or 'dequeue'")
	}

	if _, cdnExists, err := dbhelpers.GetCDNNameFromID(tx, reqObj.CDNID); err != nil {
		errorMap["cdnId"] = fmt.Errorf("could not check whether cdn exists for id %d", reqObj.CDNID)
	} else if !cdnExists {
		errorMap["cdnId"] = fmt.Errorf("no cdn exists with id %d", reqObj.CDNID)
	}

	if topologyExists, err := dbhelpers.TopologyExists(tx, string(topologyName)); err != nil {
		errorMap["topology"] = fmt.Errorf("could not check whether topology %s exists", topologyName)
	} else if !topologyExists {
		errorMap["topology"] = fmt.Errorf("no topology exists by the name of %s", topologyName)
	}

	return util.JoinErrs(tovalidate.ToErrors(errorMap))
}

// QueueUpdateHandler queues server updates for all servers in all cachegroups included in a given topology.
func QueueUpdateHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, []string{})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	reqObj := tc.TopologiesQueueUpdateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&reqObj); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	topologyName := tc.TopologyName(inf.Params["name"])
	if err := Validate(reqObj, topologyName, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, fmt.Errorf("invalid request to queue updates: %s", err), nil)
		return
	}
	if err := queueUpdates(inf.Tx.Tx, topologyName, reqObj.CDNID, reqObj.Action == "queue"); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("Topology queueing updates: "+err.Error()))
		return
	}

	message := fmt.Sprintf("TOPOLOGY: %s, ACTION: Topology server updates %sd", topologyName, reqObj.Action)
	api.CreateChangeLogRawTx(api.ApiChange, message, inf.User, inf.Tx.Tx)
	api.WriteResp(w, r, tc.TopologiesQueueUpdate{Action: reqObj.Action, CDNID: reqObj.CDNID, Topology: topologyName})
}

func queueUpdates(tx *sql.Tx, topologyName tc.TopologyName, cdnId int64, queue bool) error {
	query := `
UPDATE server s
SET upd_pending = $1
FROM cachegroup c, topology_cachegroup tc, cdn
WHERE s.cachegroup = c.id
AND c.name = tc.cachegroup
AND tc.topology = $2
AND s.cdn_id = $3
`
	var err error
	if _, err = tx.Exec(query, queue, topologyName, cdnId); err != nil {
		err = fmt.Errorf("queueing updates: %s", err)
	}
	return err
}
