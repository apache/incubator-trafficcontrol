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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/jmoiron/sqlx"
)

func Queue(w http.ResponseWriter, r *http.Request) {
	var typeID int
	var profileID int
	var ok bool
	var err error
	var str string
	queryParams := make(map[string]string, 0)

	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cols := map[string]dbhelpers.WhereColumnInfo{
		"cdnID":     {Column: "server.cdn_id", Checker: nil},
		"typeID":    {Column: "server.type", Checker: nil},
		"profileID": {Column: "server.profile", Checker: nil},
	}

	typeName := inf.Params["type"]
	profile := inf.Params["profile"]

	reqObj := tc.CDNQueueUpdateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&reqObj); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	if reqObj.Action != "queue" && reqObj.Action != "dequeue" {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("action must be 'queue' or 'dequeue'"), nil)
		return
	}
	cdnName, ok, err := dbhelpers.GetCDNNameFromID(inf.Tx.Tx, int64(inf.IntParams["id"]))
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cdn name from ID '"+inf.Params["id"]+"': "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil)
		return
	}

	// get type ID
	if typeName != "" {
		typeID, ok, err = dbhelpers.GetTypeIDByName(typeName, inf.Tx.Tx)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("error getting type ID from name: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no type ID found with that name"), nil)
			return
		}
		queryParams["typeID"] = strconv.Itoa(typeID)
		str = fmt.Sprintf(" typeID: %d", typeID)
	}

	// get profile ID
	if profile != "" {
		profileID, ok, err = dbhelpers.GetProfileIDFromName(profile, inf.Tx.Tx)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("error getting profile ID from name: "+err.Error()))
			return
		}
		if !ok {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no profile ID found with that name"), nil)
			return
		}
		queryParams["profileID"] = strconv.Itoa(profileID)
		str = fmt.Sprintf(" profileID: %d", profileID)
	}

	userErr, sysErr, statusCode := dbhelpers.CheckIfCurrentUserHasCdnLock(inf.Tx.Tx, string(cdnName), inf.User.UserName)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, statusCode, userErr, sysErr)
		return
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(queryParams, cols)
	if len(errs) > 0 {
		errCode = http.StatusBadRequest
		userErr = util.JoinErrs(errs)
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, nil)
		return
	}

	query := `UPDATE server SET upd_pending = :upd_pending`
	query = query + where + orderBy + pagination
	queryValues["upd_pending"] = reqObj.Action == "queue"
	ok, err = queueUpdates(inf.Tx, queryValues, query)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("queueing updates: %v", err))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, fmt.Errorf("no server with the given combination found"), nil)
		return
	}

	api.CreateChangeLogRawTx(api.ApiChange, "CDN: "+string(cdnName)+", ID: "+strconv.Itoa(inf.IntParams["id"])+str+", ACTION: CDN server updates "+reqObj.Action+"d", inf.User, inf.Tx.Tx)
	api.WriteResp(w, r, tc.CDNQueueUpdateResponse{Action: reqObj.Action, CDNID: int64(inf.IntParams["id"])})
}

// queueUpdates is the helper function to queue/ dequeue updates on servers for a CDN, optionally filtered by type and/ or profile
func queueUpdates(tx *sqlx.Tx, queryValues map[string]interface{}, query string) (bool, error) {
	result, err := tx.NamedExec(query, queryValues)
	if err != nil {
		return false, errors.New("querying queue updates: " + err.Error())
	} else if rc, err := result.RowsAffected(); err != nil {
		return false, fmt.Errorf("checking rows updated: %v", err)
	} else {
		return rc > 0, nil
	}
}
