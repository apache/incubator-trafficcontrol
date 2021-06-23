// Package cdn_lock contains the CRD methods which aid in locking and unlocking CDNs.
package cdn_lock

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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

const readQuery = `SELECT username, cdn, message, soft, last_updated FROM cdn_lock`
const insertQuery = `INSERT INTO cdn_lock (username, cdn, message, soft) VALUES (:username, :cdn, :message, :soft) RETURNING username, cdn, message, soft, last_updated`
const deleteQuery = `DELETE FROM cdn_lock WHERE cdn=$1 AND username=$2 RETURNING username, cdn, message, soft, last_updated`
const deleteAdminQuery = `DELETE FROM cdn_lock WHERE cdn=$1 RETURNING username, cdn, message, soft, last_updated`

// Read is the handler for GET requests to /cdn_locks.
func Read(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cols := map[string]dbhelpers.WhereColumnInfo{
		"cdn":      {Column: "cdn_lock.cdn", Checker: nil},
		"username": {Column: "cdn_lock.username", Checker: nil},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, cols)
	if len(errs) > 0 {
		errCode = http.StatusBadRequest
		userErr = util.JoinErrs(errs)
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	cdnLock := []tc.CDNLock{}
	query := readQuery + where + orderBy + pagination
	rows, err := inf.Tx.NamedQuery(query, queryValues)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("querying cdn locks: "+err.Error()))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cLock tc.CDNLock
		if err = rows.Scan(&cLock.UserName, &cLock.CDN, &cLock.Message, &cLock.Soft, &cLock.LastUpdated); err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("scanning cdn locks: "+err.Error()))
			return
		}
		cdnLock = append(cdnLock, cLock)
	}

	api.WriteResp(w, r, cdnLock)
}

// Create is the handler for POST requests to /cdn_locks.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	tx := inf.Tx.Tx
	if inf.User == nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("couldn't get user for the current request"))
	}
	var cdnLock tc.CDNLock
	if err := json.NewDecoder(r.Body).Decode(&cdnLock); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}
	if cdnLock.Soft == nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, errors.New("field 'soft' must be present"), nil)
		return
	}
	if cdnLock.CDN == "" {
		api.HandleErr(w, r, tx, http.StatusBadRequest, errors.New("field 'cdn' must be present"), nil)
		return
	}
	cdnLock.UserName = inf.User.UserName
	resultRows, err := inf.Tx.NamedQuery(insertQuery, cdnLock)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer resultRows.Close()

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&cdnLock.UserName, &cdnLock.CDN, &cdnLock.Message, &cdnLock.Soft, &cdnLock.LastUpdated); err != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("cdn lock create: scanning locks: "+err.Error()))
			return
		}
	}
	if rowsAffected == 0 {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("cdn lock create: lock couldn't be acquired"))
		return
	}
	alerts := tc.CreateAlerts(tc.SuccessLevel, "CDN lock acquired!")
	api.WriteAlertsObj(w, r, http.StatusCreated, alerts, cdnLock)

	changeLogMsg := fmt.Sprintf("USER: %s, CDN: %s, ACTION: Lock Acquired", inf.User.UserName, cdnLock.CDN)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}

// Delete is the handler for DELETE requests to /cdn_locks.
func Delete(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"cdn"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cdn := inf.Params["cdn"]
	tx := inf.Tx.Tx
	if inf.User == nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("couldn't get user for the current request"))
	}
	var result tc.CDNLock
	var err error
	if inf.User.PrivLevel == auth.PrivLevelAdmin {
		err = inf.Tx.Tx.QueryRow(deleteAdminQuery, cdn).Scan(&result.UserName, &result.CDN, &result.Message, &result.Soft, &result.LastUpdated)
	} else {
		err = inf.Tx.Tx.QueryRow(deleteQuery, cdn, inf.User.UserName).Scan(&result.UserName, &result.CDN, &result.Message, &result.Soft, &result.LastUpdated)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if inf.User.PrivLevel != auth.PrivLevelAdmin {
				api.HandleErr(w, r, tx, http.StatusForbidden, fmt.Errorf("deleting cdn lock with cdn name %s: operation forbidden", cdn), nil)
				return
			}
			api.HandleErr(w, r, tx, http.StatusNotFound, fmt.Errorf("deleting cdn lock with cdn name %s: lock not found", cdn), nil)
			return
		}
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("deleting cdn lock with cdn name %s : %w", cdn, err))
		return
	}
	alerts := tc.CreateAlerts(tc.SuccessLevel, "cdn lock deleted")
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, result)
	changeLogMsg := fmt.Sprintf("USER: %s, CDN: %s, ACTION: Lock Released", result.UserName, cdn)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)
}
