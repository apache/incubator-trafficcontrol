package region

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
)

func GetName(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	regionNames, err := getName(inf.Tx.Tx, inf.Params["name"])

	deprecationMessage := "GET /regions?name={{name}}"
	alerts := api.CreateDeprecationAlerts(&deprecationMessage)
	if err != nil {
		alerts.AddNewAlert(tc.ErrorLevel, err.Error())
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, regionNames)
}

// getName returns a slice, even though only 1 region will ever be returned, because that's what the 1.x API responds with.
func getName(tx *sql.Tx, name string) ([]tc.RegionName, error) {
	r := tc.RegionName{Name: name}
	err := tx.QueryRow(`SELECT r.id, d.id, d.name FROM region as r JOIN division as d ON r.division = d.id WHERE r.name = $1`, name).Scan(&r.ID, &r.Division.ID, &r.Division.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return []tc.RegionName{}, nil
		}
		return nil, errors.New("querying region by name: " + err.Error())
	}
	return []tc.RegionName{r}, nil
}
