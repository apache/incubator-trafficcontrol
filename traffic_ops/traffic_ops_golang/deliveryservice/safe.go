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
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

const safeUpdateQuery = `
UPDATE deliveryservice
SET display_name=$1,
    info_url=$2,
    long_desc=$3,
    long_desc_1=$4
WHERE id = $5
RETURNING id
`

// UpdateSafe is the handler for PUT requests to /deliveryservices/{{ID}}/safe.
//
// The only fields which are "safe" to modify are the displayName, infoURL, longDesc, and longDesc1.
func UpdateSafe(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	dsID := inf.IntParams["id"]

	userErr, sysErr, errCode = tenant.CheckID(tx, inf.User, dsID)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	var dsr tc.DeliveryServiceSafeUpdateRequest
	if err := api.Parse(r.Body, tx, &dsr); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, fmt.Errorf("decoding: %s", err), nil)
		return
	}
	if ok, err := updateDSSafe(tx, dsID, dsr); err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("Updating Delivery Service (safe): %s", err))
		return
	} else if !ok {
		userErr = fmt.Errorf("No Delivery Service exists by ID '%d'", dsID)
		api.HandleErr(w, r, tx, http.StatusNotFound, userErr, nil)
		return
	}
	useIMS := false
	config, e := api.GetConfig(r.Context())
	if e == nil && config != nil {
		useIMS = config.UseIMS
	} else {
		log.Warnf("Couldn't get config %v", e)
	}
	dses, userErr, sysErr, errCode, _ := readGetDeliveryServices(r.Header, inf, useIMS)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	if len(dses) != 1 {
		sysErr = fmt.Errorf("Updating Delivery Service (safe): expected one Delivery Service returned from read, got %v", len(dses))
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}

	ds := dses[0]

	alertMsg := "Delivery Service safe update successful."
	if inf.Version != nil && inf.Version.Major == 1 && inf.Version.Minor < 5 {
		switch inf.Version.Minor {
		case 4:
			api.WriteRespAlertObj(w, r, tc.SuccessLevel, alertMsg, []tc.DeliveryServiceNullableV14{ds.DowngradeToV3().DeliveryServiceNullableV14})
		case 3:
			api.WriteRespAlertObj(w, r, tc.SuccessLevel, alertMsg, []tc.DeliveryServiceNullableV13{ds.DowngradeToV3().DeliveryServiceNullableV13})
		default:
			api.WriteRespAlertObj(w, r, tc.SuccessLevel, alertMsg, []tc.DeliveryServiceNullableV12{ds.DowngradeToV3().DeliveryServiceNullableV12})
		}
	} else {
		api.WriteRespAlertObj(w, r, tc.SuccessLevel, alertMsg, []tc.DeliveryServiceNullableV30{ds.DowngradeToV3()})
	}

	api.CreateChangeLogRawTx(api.ApiChange, fmt.Sprintf("DS: %s, ID: %d, ACTION: Updated safe fields", ds.XMLID, *ds.ID), inf.User, tx)
}

// updateDSSafe updates the given delivery service in the database. Returns whether the DS existed, and any error.
func updateDSSafe(tx *sql.Tx, dsID int, ds tc.DeliveryServiceSafeUpdateRequest) (bool, error) {
	res, err := tx.Exec(safeUpdateQuery, ds.DisplayName, ds.InfoURL, ds.LongDesc, ds.LongDesc1, dsID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("Checking rows affected: %s", err)
	}
	if rowsAffected < 1 {
		return false, nil
	}
	if rowsAffected > 1 {
		return false, fmt.Errorf("Too many rows affected: %v", rowsAffected)
	}
	return true, nil
}
