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

// GetAssignment is the handler for GET requests to
// /deliveryservice_requests/{{ID}}/assign.
func GetAssignment(w http.ResponseWriter, r *http.Request) {
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

	api.WriteResp(w, r, dsr.Assignee)
}

type assignmentRequest struct {
	AssigneeID *int    `json:"assigneeId"`
	Assignee   *string `json:"assignee"`
}

func (*assignmentRequest) Validate(*sql.Tx) error {
	return nil
}

const assignDSRQuery = `
UPDATE deliveryservice_request
SET assignee_id = $1
WHERE id = $2
RETURNING last_updated
`

func getAssignee(r *assignmentRequest, xmlID string, tx *sql.Tx) (string, int, error, error) {
	if r == nil || tx == nil {
		return "", http.StatusInternalServerError, nil, errors.New("nil transaction or assignment request")
	}

	var message string
	if r.AssigneeID != nil {
		r.Assignee = new(string)
		if err := tx.QueryRow(`SELECT username FROM tm_user WHERE id = $1`, *r.AssigneeID).Scan(r.Assignee); err == sql.ErrNoRows {
			userErr := fmt.Errorf("no such user #%d", *r.AssigneeID)
			return "", http.StatusBadRequest, userErr, nil
		} else if err != nil {
			sysErr := fmt.Errorf("getting username for assignee ID (#%d): %v", *r.AssigneeID, err)
			return "", http.StatusInternalServerError, nil, sysErr
		}
		message = fmt.Sprintf("Changed assignee of '%s' Delivery Service Request to '%s'", xmlID, *r.Assignee)
	} else if r.Assignee != nil {
		r.AssigneeID = new(int)
		if err := tx.QueryRow(`SELECT id FROM tm_user WHERE username=$1`, *r.Assignee).Scan(r.AssigneeID); err == sql.ErrNoRows {
			userErr := fmt.Errorf("no such user '%s'", *r.Assignee)
			return "", http.StatusBadRequest, userErr, nil
		} else if err != nil {
			sysErr := fmt.Errorf("getting user ID for assignee (%s): %v", *r.Assignee, err)
			return "", http.StatusInternalServerError, nil, sysErr
		}
		message = fmt.Sprintf("Changed assignee of '%s' Delivery Service Request to '%s'", xmlID, *r.Assignee)
	} else {
		message = fmt.Sprintf("Unassigned '%s' Delivery Service Request", xmlID)
	}
	return message, http.StatusOK, nil, nil
}

// PutAssignment is the handler for PUT requsets to
// /deliveryservice_requests/{{ID}}/assign.
func PutAssignment(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var req assignmentRequest
	if err := api.Parse(r.Body, tx, &req); err != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, err, nil)
		return
	}

	// Don't accept "assignee" in lieu of "assigneeId" in API version < 4.0
	if inf.Version.Major < 4 {
		req.Assignee = nil
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
	dsr.SetXMLID()

	authorized, err := isTenantAuthorized(dsr, inf)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !authorized {
		api.HandleErr(w, r, tx, http.StatusForbidden, errors.New("not authorized on this tenant"), nil)
		return
	}

	message, errCode, userErr, sysErr := getAssignee(&req, dsr.XMLID, tx)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	if err := tx.QueryRow(assignDSRQuery, req.AssigneeID, *dsr.ID).Scan(&dsr.LastUpdated); err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	dsr.Assignee = req.Assignee
	dsr.AssigneeID = req.AssigneeID

	if dsr.ChangeType == tc.DSRChangeTypeUpdate {
		query := deliveryservice.SelectDeliveryServicesQuery + `WHERE xml_id=:XMLID`
		originals, userErr, sysErr, errCode := deliveryservice.GetDeliveryServices(query, map[string]interface{}{"XMLID": dsr.XMLID}, inf.Tx)
		if userErr != nil || sysErr != nil {
			api.HandleErr(w, r, tx, errCode, userErr, sysErr)
			return
		}
		if len(originals) < 1 {
			userErr = fmt.Errorf("cannot update non-existent Delivery Service '%s'", dsr.XMLID)
			api.HandleErr(w, r, tx, http.StatusBadRequest, userErr, nil)
			return
		}
		if len(originals) > 1 {
			sysErr = fmt.Errorf("too many Delivery Services with XMLID '%s'; want: 1, got: %d", dsr.XMLID, len(originals))
			api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, sysErr)
			return
		}
		dsr.Original = new(tc.DeliveryServiceV4)
		*dsr.Original = originals[0]
	}
	var resp interface{}
	if inf.Version.Major >= 4 {
		resp = dsr
	} else {
		resp = dsr.Downgrade()
	}

	api.WriteRespAlertObj(w, r, tc.SuccessLevel, message, resp)
	// This matches the CRUDer changelog format. Note, though, that it
	// references the DSR's ID three times and names the affected table
	// twice. Lotta redundancy - so might be worth changing?
	message = fmt.Sprintf("Delivery Service Request: %d, ID: %d, ACTION: %s deliveryservice_request, keys: {id:%d }", *dsr.ID, *dsr.ID, message, *dsr.ID)
	inf.CreateChangeLog(message)
}
