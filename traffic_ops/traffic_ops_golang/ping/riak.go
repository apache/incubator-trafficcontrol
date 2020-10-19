package ping

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
	"errors"
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
)

func Riak(w http.ResponseWriter, r *http.Request) {
	alerts := tc.CreateAlerts(tc.WarnLevel, fmt.Sprintf("This endpoint is deprecated, please use GET /api/2.0/vault/ping instead"))
	inf, errs := api.NewInfo(r, nil, nil)

	if errs.Occurred() {
		// TODO: I think this should just be using the deprecated error handler
		userErr := api.LogErrs(r, errs)
		alerts.AddAlerts(tc.CreateErrorAlerts(userErr))
		api.WriteAlerts(w, r, errs.Code, alerts)
		return
	}

	defer inf.Close()

	pingResp, err := riaksvc.Ping(inf.Tx.Tx, inf.Config.RiakAuthOptions, inf.Config.RiakPort)

	if err != nil {
		userErr := api.LogErr(r, http.StatusInternalServerError, nil, errors.New("error pinging Riak: "+err.Error()))
		alerts.AddAlerts(tc.CreateErrorAlerts(userErr))
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, pingResp)
}
