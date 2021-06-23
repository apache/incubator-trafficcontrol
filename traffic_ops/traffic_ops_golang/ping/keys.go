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
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

const API_VAULT_PING = "/vault/ping"

func Keys(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleDeprecatedErr(w, r, nil, errCode, userErr, sysErr, util.StrPtr(API_VAULT_PING))
		return
	}
	defer inf.Close()

	pingResp, err := inf.Vault.Ping(inf.Tx.Tx, r.Context())
	if err != nil {
		api.HandleDeprecatedErr(w, r, nil, http.StatusInternalServerError, err, nil, util.StrPtr(API_VAULT_PING))
		return
	}
	api.WriteAlertsObj(w, r, http.StatusOK, api.CreateDeprecationAlerts(util.StrPtr(API_VAULT_PING)), pingResp)
}
