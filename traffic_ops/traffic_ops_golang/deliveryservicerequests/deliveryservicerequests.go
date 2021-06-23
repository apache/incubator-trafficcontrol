// Package deliveryservicerequests contains the handler for the endpoint
// responsible for issuing a request for Delivery Service creation via email -
// not to be confused with package
// github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/deliveryservice/request.
//
// Deprecated: This package contains logic relating to a deprecated API
// endpoint and is subject to removal in the future.
package deliveryservicerequests

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
	"fmt"
	"net/http"
	"net/mail"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

const msg = "From: %s\r\nTo:%s\r\nContent-Type: text/html\r\nSubject: Delivery Service Request for %s\r\n\r\n%s"

// Request is the handler for creating a Delivery Service request via
// /deliveryservices/request - not to be confused with a Delivery Service
// Request as created using the /deliveryservice_requests.
func Request(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var dsr tc.DeliveryServiceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&dsr); err != nil {
		userErr = fmt.Errorf("parsing request: %w", err)
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	if err := dsr.Validate(); err != nil {
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, err, nil)
		return
	}

	addr, err := mail.ParseAddress(dsr.EmailTo)
	if err != nil {
		userErr = fmt.Errorf("'%s' is not a valid RFC5322 email address", dsr.EmailTo)
		sysErr = fmt.Errorf("parsing submitted email address: %w", err)
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	body, err := dsr.Details.Format()
	if err != nil {
		sysErr = fmt.Errorf("failed to format email body: %w", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	body = fmt.Sprintf(msg, inf.Config.ConfigTO.EmailFrom, addr, dsr.Details.Customer, body)
	errCode, userErr, sysErr = inf.SendMail(rfc.EmailAddress{Address: *addr}, []byte(body))
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	var alerts tc.Alerts
	alerts.AddNewAlert(tc.SuccessLevel, "Delivery Service request sent to "+dsr.EmailTo)
	alerts.AddNewAlert(tc.WarnLevel, "This endpoint is deprecated, it has been removed in API versions 4.0 and later")
	api.WriteAlerts(w, r, http.StatusOK, alerts)
}
