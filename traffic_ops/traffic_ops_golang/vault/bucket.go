package vault

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
	"errors"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

// Deprecated
func GetBucketKeyDeprecated(w http.ResponseWriter, r *http.Request) {
	getBucketKey(w, r, api.CreateDeprecationAlerts(nil))
}

func getBucketKey(w http.ResponseWriter, r *http.Request, a tc.Alerts) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"bucket", "key"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if !inf.Config.TrafficVaultEnabled {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, userErr, errors.New("getting bucket key: Traffic Vault is not configured"))
		return
	}

	val, ok, err := inf.Vault.GetBucketKey(inf.Params["bucket"], inf.Params["key"], inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting bucket key from Traffic Vault: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("not found"), nil)
		return
	}

	valObj := map[string]interface{}{}
	if err := json.Unmarshal(val, &valObj); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("GetBucketKey bucket '"+inf.Params["bucket"]+"' key '"+inf.Params["key"]+"' Traffic Vault returned invalid JSON: "+err.Error()))
		return
	}

	if len(a.Alerts) > 0 {
		api.WriteAlertsObj(w, r, http.StatusOK, a, valObj)
	} else {
		api.WriteResp(w, r, valObj)
	}

}
