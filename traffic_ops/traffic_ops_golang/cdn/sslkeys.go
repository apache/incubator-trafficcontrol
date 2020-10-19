package cdn

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"

	"github.com/basho/riak-go-client"
)

func GetSSLKeys(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, []string{"name"}, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()
	keys, err := getSSLKeys(inf.Tx.Tx, inf.Config.RiakAuthOptions, inf.Config.RiakPort, inf.Params["name"])
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting cdn ssl keys: "+err.Error()))
		return
	}
	api.WriteResp(w, r, keys)
}

func getSSLKeys(tx *sql.Tx, authOpts *riak.AuthOptions, riakPort *uint, cdnName string) ([]tc.CDNSSLKey, error) {
	keys, err := riaksvc.GetCDNSSLKeysObj(tx, authOpts, riakPort, cdnName)
	if err != nil {
		return nil, errors.New("getting cdn ssl keys from Riak: " + err.Error())
	}
	return keys, nil
}
