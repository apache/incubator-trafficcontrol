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
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
)

func GetBucketKeyDeprecated(w http.ResponseWriter, r *http.Request) {
	getBucketKey(w, r, api.CreateDeprecationAlerts(util.StrPtr("/vault/bucket/:bucket/key/:key/values")))
}

func GetBucketKey(w http.ResponseWriter, r *http.Request) {
	getBucketKey(w, r, tc.Alerts{})
}

func getBucketKey(w http.ResponseWriter, r *http.Request, a tc.Alerts) {
	inf, errs := api.NewInfo(r, []string{"bucket", "key"}, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	if inf.Config.RiakEnabled == false {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("riak.GetBucketKey: Riak is not configured!"))
		return
	}

	val, ok, err := riaksvc.GetBucketKey(inf.Tx.Tx, inf.Config.RiakAuthOptions, inf.Config.RiakPort, inf.Params["bucket"], inf.Params["key"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting bucket key from Riak: "+err.Error()))
		return
	}
	if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("not found"), nil)
		return
	}

	valObj := map[string]interface{}{}
	if err := json.Unmarshal(val, &valObj); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("GetBucketKey bucket '"+inf.Params["bucket"]+"' key '"+inf.Params["key"]+"' Riak returned invalid JSON: "+err.Error()))
		return
	}

	if len(a.Alerts) > 0 {
		api.WriteAlertsObj(w, r, http.StatusOK, a, valObj)
	} else {
		api.WriteResp(w, r, valObj)
	}

}
