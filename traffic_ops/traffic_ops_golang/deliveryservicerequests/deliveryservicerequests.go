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

func Request(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	tx := inf.Tx.Tx
	var dsr tc.DeliveryServiceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&dsr); err != nil {
		userErr := fmt.Errorf("Error parsing request: %v", err)
		errCode := http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	if err := dsr.Validate(); err != nil {
		errCode := http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, err, nil)
		return
	}

	addr, err := mail.ParseAddress(dsr.EmailTo)
	if err != nil {
		userErr := fmt.Errorf("'%s' is not a valid RFC5322 email address!", dsr.EmailTo)
		sysErr := fmt.Errorf("Parsing submitted email address: %s", err)
		errCode := http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	body, err := dsr.Details.Format()
	if err != nil {
		sysErr := fmt.Errorf("Failed to format email body: %v", err)
		errCode := http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	body = fmt.Sprintf(msg, inf.Config.ConfigTO.EmailFrom, addr, dsr.Details.Customer, body)
	errs = inf.SendMail(rfc.EmailAddress{*addr}, []byte(body))
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}

	api.WriteRespAlert(w, r, tc.SuccessLevel, fmt.Sprintf("Delivery Service request sent to %s", dsr.EmailTo))
	w.Write([]byte{'\n'})
}
