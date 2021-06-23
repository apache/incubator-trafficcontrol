package crstats

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
	"github.com/lib/pq"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

// GetDSRouting is the handler for getting aggregated routing percentages for a DS.
func GetDSRouting(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx

	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	dsID := inf.IntParams["id"]

	userErr, sysErr, errCode = tenant.CheckID(tx, inf.User, dsID)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	dsType, exists, err := dbhelpers.GetDeliveryServiceType(dsID, tx)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("getting delivery service type: %v", err))
		return
	}
	if !exists {
		api.HandleErr(w, r, tx, http.StatusNotFound, fmt.Errorf("delivery service %v not found", dsID), nil)
		return
	}

	stat := ""
	if tc.DSType(dsType).IsHTTP() {
		stat = HTTP
	} else if tc.DSType(dsType).IsDNS() {
		stat = DNS
	} else {
		api.HandleErr(w, r, tx, http.StatusBadRequest, fmt.Errorf("delivery service %v does not have a HTTP or DNS type", dsID), nil)
		return
	}

	q := `SELECT ARRAY(SELECT r.pattern FROM deliveryservice_regex dsr JOIN regex r ON dsr.regex = r.id JOIN type t ON r.type = t.id WHERE t.name = 'HOST_REGEXP' AND dsr.deliveryservice = $1)`
	patterns := []string{}
	err = tx.QueryRow(q, dsID).Scan(pq.Array(&patterns))
	if err != nil {
		if err != sql.ErrNoRows {
			api.HandleErr(w, r, tx, http.StatusBadRequest, fmt.Errorf("delivery service %v does not have host regexps to match routing stats to", dsID), nil)
			return
		}
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("querying delivery service %v patterns - %v", dsID, err))
		return
	}

	routers, err := getCDNRouterFQDNs(tx, nil)
	if err != nil {
		api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, errors.New("getting monitors: "+err.Error()))
		return
	}

	api.RespWriter(w, r, inf.Tx.Tx)(getRoutersRouting(tx, routers, &stat, patterns))
}
