package servers

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

// Delete handler for deleting the association between a Delivery Service and a Server
func Delete(w http.ResponseWriter, r *http.Request) {
	delete(w, r, false)
}

// Delete deprecatation handler for deleting the association between a Delivery Service and a Server
func DeleteDeprecated(w http.ResponseWriter, r *http.Request) {
	delete(w, r, true)
}

func delete(w http.ResponseWriter, r *http.Request, deprecated bool) {
	alt := "DELETE deliveryserviceserver/:dsid/:serverid"
	inf, errs := api.NewInfo(r, []string{"serverid", "dsid"}, []string{"serverid", "dsid"})
	if errs.Occurred() {
		api.HandleErrsOptionalDeprecation(w, r, inf.Tx.Tx, errs, deprecated, &alt)
		return
	}
	defer inf.Close()

	dsID := inf.IntParams["dsid"]
	serverID := inf.IntParams["serverid"]

	errs = tenant.CheckID(inf.Tx.Tx, inf.User, dsID)
	if errs.Occurred() {
		api.HandleErrsOptionalDeprecation(w, r, inf.Tx.Tx, errs, deprecated, &alt)
		return
	}

	dsName, _, err := dbhelpers.GetDSNameFromID(inf.Tx.Tx, dsID)
	if err != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting delivery service name from id: "+err.Error()), deprecated, &alt)
		return
	}

	serverName, exists, err := dbhelpers.GetServerNameFromID(inf.Tx.Tx, serverID)
	if err != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting server name from id: "+err.Error()), deprecated, &alt)
		return
	} else if !exists {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("server not found"), nil, deprecated, &alt)
		return
	}

	ok, err := deleteDSServer(inf.Tx.Tx, dsID, serverID)
	if err != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deleting delivery service server: "+err.Error()), deprecated, &alt)
		return
	}
	if !ok {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusNotFound, nil, nil, deprecated, &alt)
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "DS: "+string(dsName)+", ID: "+strconv.Itoa(dsID)+", ACTION: Remove server "+string(serverName)+" from delivery service", inf.User, inf.Tx.Tx)
	if deprecated {
		alerts := api.CreateDeprecationAlerts(&alt)
		alerts.AddNewAlert(tc.SuccessLevel, "Server unlinked from delivery service.")
		api.WriteAlerts(w, r, http.StatusOK, alerts)
		return
	}
	api.WriteRespAlert(w, r, tc.SuccessLevel, "Server unlinked from delivery service.")
}

// deleteDSServer deletes the given deliveryservice_server. Returns whether the server existed, and any error.
func deleteDSServer(tx *sql.Tx, dsID int, serverID int) (bool, error) {
	deletedServerID := 0
	if err := tx.QueryRow(`DELETE FROM deliveryservice_server WHERE deliveryservice = $1 AND server = $2 RETURNING server`, dsID, serverID).Scan(&deletedServerID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.New("deleting delivery service server: " + err.Error())
	}
	return true, nil
}
