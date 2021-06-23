package profile

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

func Trimmed(w http.ResponseWriter, r *http.Request) {
	alt := "GET /profiles"
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, &alt)
		return
	}
	defer inf.Close()
	trimmed, err := getTrimmedProfiles(inf.Tx.Tx)
	if err != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, &alt)
		return
	}
	alerts := api.CreateDeprecationAlerts(&alt)
	api.WriteAlertsObj(w, r, http.StatusOK, alerts, trimmed)
}

func getTrimmedProfiles(tx *sql.Tx) ([]tc.ProfileTrimmed, error) {
	rows, err := tx.Query(`SELECT name FROM profile`)
	if err != nil {
		return nil, errors.New("querying trimmed profiles: " + err.Error())
	}
	defer rows.Close()
	profiles := []tc.ProfileTrimmed{}
	for rows.Next() {
		name := ""
		if err = rows.Scan(&name); err != nil {
			return nil, errors.New("scanning trimmed profiles: " + err.Error())
		}
		profiles = append(profiles, tc.ProfileTrimmed{Name: name})
	}
	return profiles, nil
}
