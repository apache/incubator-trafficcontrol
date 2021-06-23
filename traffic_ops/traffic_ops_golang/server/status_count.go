package server

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

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

func GetServersStatusCountsHandler(w http.ResponseWriter, r *http.Request) {
	alt := "GET /servers"
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr, &alt)
		return
	}
	defer inf.Close()

	statusCounts, err := getServersStatusCounts(inf.Tx.Tx, inf.Params["type"])
	if err != nil {
		api.HandleDeprecatedErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting servers status counts: "+err.Error()), &alt)
		return
	}
	api.WriteAlertsObj(w, r, http.StatusOK, api.CreateDeprecationAlerts(&alt), statusCounts)
}

func getServersStatusCounts(tx *sql.Tx, typeName string) (map[string]int, error) {
	where := ""
	args := make([]interface{}, 0, 1)
	if typeName != "" {
		where = "WHERE type.name = $1"
		args = append(args, typeName)
	}
	q := `
SELECT status.name, count(server.id)
FROM server
JOIN status ON server.status = status.id
JOIN type ON server.type = type.id
` + where + `
GROUP BY status.id
`
	rows, err := tx.Query(q, args...)
	if err != nil {
		return nil, errors.New("querying server status counts: " + err.Error())
	}
	defer rows.Close()
	statusCounts := map[string]int{}
	for rows.Next() {
		statusName := ""
		count := 0
		if err := rows.Scan(&statusName, &count); err != nil {
			return nil, errors.New("scanning server status counts: " + err.Error())
		}
		statusCounts[statusName] = count
	}
	return statusCounts, nil
}
