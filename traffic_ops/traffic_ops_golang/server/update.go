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
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

// UpdateHandler implements an http handler that updates a server's upd_pending and reval_pending values.
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id-or-name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	idOrName := inf.Params["id-or-name"]
	id, err := strconv.Atoi(idOrName)
	hostName := ""
	if err == nil {
		name, ok, err := dbhelpers.GetServerNameFromID(inf.Tx.Tx, id)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting server name from id '"+idOrName+"': "+err.Error()))
			return
		} else if !ok {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("server ID '"+idOrName+"' not found"), nil)
			return
		}
		hostName = name
	} else {
		hostName = idOrName
		if _, ok, err := dbhelpers.GetServerIDFromName(hostName, inf.Tx.Tx); err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting server id from name '"+idOrName+"': "+err.Error()))
			return
		} else if !ok {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("server name '"+idOrName+"' not found"), nil)
			return
		}
	}

	updated, hasUpdated := inf.Params["updated"]
	revalUpdated, hasRevalUpdated := inf.Params["reval_updated"]
	if !hasUpdated && !hasRevalUpdated {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("Must pass at least one query paramter of 'updated' or 'reval_updated'"), nil)
		return
	}
	updated = strings.ToLower(updated)
	revalUpdated = strings.ToLower(revalUpdated)

	if hasUpdated && updated != `t` && updated != `true` && updated != `f` && updated != `false` {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("query parameter 'updated' must be 'true' or 'false'"), nil)
		return
	}
	if hasRevalUpdated && revalUpdated != `t` && revalUpdated != `true` && revalUpdated != `f` && revalUpdated != `false` {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("query parameter 'reval_updated' must be 'true' or 'false'"), nil)
		return
	}

	strToBool := func(s string) bool {
		return !strings.HasPrefix(strings.ToLower(s), "f")
	}

	updatedPtr := (*bool)(nil)
	if hasUpdated {
		updatedBool := strToBool(updated)
		updatedPtr = &updatedBool
	}
	revalUpdatedPtr := (*bool)(nil)
	if hasRevalUpdated {
		revalUpdatedBool := strToBool(revalUpdated)
		revalUpdatedPtr = &revalUpdatedBool
	}

	if err := setUpdateStatuses(inf.Tx.Tx, hostName, updatedPtr, revalUpdatedPtr); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("setting updated statuses: "+err.Error()))
		return
	}

	respMsg := "successfully set server '" + hostName + "'"
	if hasUpdated {
		respMsg += " updated=" + strconv.FormatBool(strToBool(updated))
	}
	if hasRevalUpdated {
		respMsg += " reval_updated=" + strconv.FormatBool(strToBool(revalUpdated))
	}

	api.WriteAlerts(w, r, http.StatusOK, tc.CreateAlerts(tc.SuccessLevel, respMsg))
}

// setUpdateStatuses sets the upd_pending and reval_pending columns of a server.
// If updatePending or revalPending is nil, that value is not changed.
func setUpdateStatuses(tx *sql.Tx, hostName string, updatePending *bool, revalPending *bool) error {
	if updatePending == nil && revalPending == nil {
		return errors.New("either updatePending or revalPending must not be nil")
	}
	qry := `UPDATE server SET `
	updateStrs := []string{}
	nextI := 1
	qryVals := []interface{}{}
	if updatePending != nil {
		updateStrs = append(updateStrs, `upd_pending = $`+strconv.Itoa(nextI))
		nextI++
		qryVals = append(qryVals, *updatePending)
	}
	if revalPending != nil {
		updateStrs = append(updateStrs, `reval_pending = $`+strconv.Itoa(nextI))
		nextI++
		qryVals = append(qryVals, *revalPending)
	}
	qry += strings.Join(updateStrs, ", ") + ` WHERE host_name = $` + strconv.Itoa(nextI)
	qryVals = append(qryVals, hostName)

	if _, err := tx.Exec(qry, qryVals...); err != nil {
		return errors.New("executing: " + err.Error())
	}
	return nil
}
