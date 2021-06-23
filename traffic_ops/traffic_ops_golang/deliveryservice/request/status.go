package request

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
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice"
)

// GetStatus is the handler for GET requests to
// /deliveryservice_requests/{{ID}}/status.
func GetStatus(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	// This should never happen because a route doesn't exist for it
	if inf.Version.Major < 4 {
		w.Header().Set("Allow", http.MethodPut)
		w.WriteHeader(http.StatusMethodNotAllowed)
		api.WriteRespAlert(w, r, tc.ErrorLevel, http.StatusText(http.StatusMethodNotAllowed))
		return
	}

	var dsr tc.DeliveryServiceRequestV40
	if err := inf.Tx.QueryRowx(selectQuery+"WHERE r.id=$1", inf.IntParams["id"]).StructScan(&dsr); err != nil {
		if err == sql.ErrNoRows {
			errCode = http.StatusNotFound
			userErr = fmt.Errorf("no such Delivery Service Request: %d", inf.IntParams["id"])
			sysErr = nil
		} else {
			errCode = http.StatusInternalServerError
			userErr = nil
			sysErr = fmt.Errorf("looking for DSR: %v", err)
		}
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	authorized, err := isTenantAuthorized(dsr, inf)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !authorized {
		api.HandleErr(w, r, tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	api.WriteResp(w, r, dsr.Status)
}

const updateStatusQuery = `
UPDATE deliveryservice_request
SET status = $1, last_edited_by_id = $2
WHERE id = $3
RETURNING last_updated
`

const updateStatusAndOriginalQuery = `
UPDATE deliveryservice_request
SET original=$1, status=$2, last_edited_by_id=$3
WHERE id=$4
RETURNING last_updated
`

// PutStatus is the handler for PUT requests to
// /deliveryservice_requests/{{ID}}/status.
func PutStatus(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.User == nil {
		sysErr = errors.New("got api info with no user")
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
		return
	}

	var req tc.StatusChangeRequest
	if err := api.Parse(r.Body, tx, &req); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	dsrID := inf.IntParams["id"]

	var dsr tc.DeliveryServiceRequestV40
	if err := inf.Tx.QueryRowx(selectQuery+"WHERE r.id=$1", dsrID).StructScan(&dsr); err != nil {
		if err == sql.ErrNoRows {
			errCode = http.StatusNotFound
			userErr = fmt.Errorf("no such Delivery Service Request: %d", dsrID)
			sysErr = nil
		} else {
			errCode = http.StatusInternalServerError
			userErr = nil
			sysErr = fmt.Errorf("looking for DSR: %v", err)
		}
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	dsr.SetXMLID()

	if err := dsr.Status.ValidTransition(req.Status); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	authorized, err := isTenantAuthorized(dsr, inf)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !authorized {
		api.HandleErr(w, r, tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	dsr.LastEditedBy = inf.User.UserName
	dsr.LastEditedByID = new(int)
	*dsr.LastEditedByID = inf.User.ID

	// store the current original DS if the DSR is being closed
	// (and isn't a "create" request)
	if dsr.IsOpen() && req.Status != tc.RequestStatusDraft && req.Status != tc.RequestStatusSubmitted && dsr.ChangeType != tc.DSRChangeTypeCreate {
		if dsr.ChangeType == tc.DSRChangeTypeUpdate && dsr.Requested != nil && dsr.Requested.ID != nil {
			errCode, userErr, sysErr = getOriginals([]int{*dsr.Requested.ID}, inf.Tx, map[int][]*tc.DeliveryServiceRequestV4{*dsr.Requested.ID: {&dsr}})
			if userErr != nil || sysErr != nil {
				api.HandleErr(w, r, tx, errCode, userErr, sysErr)
				return
			}
			if dsr.Original == nil {
				sysErr = fmt.Errorf("failed to build original from dsr #%d that was to be closed; requested ID: %d", dsrID, *dsr.Requested.ID)
			}
		} else if dsr.ChangeType == tc.DSRChangeTypeDelete && dsr.Original != nil && dsr.Original.ID != nil {
			errCode, userErr, sysErr = getOriginals([]int{*dsr.Original.ID}, inf.Tx, map[int][]*tc.DeliveryServiceRequestV4{*dsr.Original.ID: {&dsr}})
			if userErr != nil || sysErr != nil {
				api.HandleErr(w, r, tx, errCode, userErr, sysErr)
				return
			}
			if dsr.Original == nil {
				sysErr = fmt.Errorf("failed to build original from dsr #%d that was to be closed; original ID: %d", dsrID, *dsr.Original.ID)
			}
		}

		if sysErr != nil {
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}

		err := tx.QueryRow(updateStatusAndOriginalQuery, dsr.Original, req.Status, dsr.LastEditedByID, dsrID).Scan(&dsr.LastUpdated)
		if err != nil {
			sysErr = fmt.Errorf("updating original for dsr #%d: %v", dsrID, err)
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
	} else if err := tx.QueryRow(updateStatusQuery, req.Status, dsr.LastEditedByID, *dsr.ID).Scan(&dsr.LastUpdated); err == nil {
		if dsr.IsOpen() && dsr.ChangeType != tc.DSRChangeTypeCreate {
			query := deliveryservice.SelectDeliveryServicesQuery + " WHERE ds.xml_id = :xmlid"
			original, userErr, sysErr, errCode := deliveryservice.GetDeliveryServices(query, map[string]interface{}{"xmlid": dsr.XMLID}, inf.Tx)
			if userErr != nil || sysErr != nil {
				api.HandleErr(w, r, tx, errCode, userErr, sysErr)
				return
			}
			if len(original) != 1 {
				api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("expected exactly one DS with XMLID '%s', found: %d", dsr.XMLID, len(original)))
				return
			}
			dsr.Original = new(tc.DeliveryServiceV4)
			*dsr.Original = original[0]
		}
	} else {
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	message := fmt.Sprintf("Changed status of '%s' Delivery Service Request from '%s' to '%s'", dsr.XMLID, dsr.Status, req.Status)
	dsr.Status = req.Status

	var resp interface{}
	if inf.Version.Major >= 4 {
		resp = dsr
	} else {
		resp = dsr.Downgrade()
	}

	api.WriteRespAlertObj(w, r, tc.SuccessLevel, message, resp)
	message = fmt.Sprintf("Delivery Service Request: %d, ID: %d, ACTION: %s deliveryservice_request, keys: {id:%d }", *dsr.ID, *dsr.ID, message, *dsr.ID)
	inf.CreateChangeLog(message)
}
