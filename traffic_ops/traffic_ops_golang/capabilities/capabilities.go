package capabilities

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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

const readQuery = `
SELECT description,
       last_updated,
       name
FROM capability
`

const createQuery = `
INSERT INTO capability (name, description)
VALUES ($1, $2)
RETURNING description, last_updated, name
`

func Read(w http.ResponseWriter, r *http.Request) {
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	cols := map[string]dbhelpers.WhereColumnInfo{
		"name": dbhelpers.WhereColumnInfo{Column: "capability.name"},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, cols)
	if len(errs) > 0 {
		errCode = http.StatusBadRequest
		userErr = util.JoinErrs(errs)
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	query := readQuery + where + orderBy + pagination
	rows, err := inf.Tx.NamedQuery(query, queryValues)
	if err != nil && err != sql.ErrNoRows {
		errCode = http.StatusInternalServerError
		sysErr = fmt.Errorf("querying capabilities: %v", err)
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}
	defer rows.Close()

	caps := []tc.Capability{}
	for rows.Next() {
		cap := tc.Capability{}
		if err := rows.Scan(&cap.Description, &cap.LastUpdated, &cap.Name); err != nil {
			errCode = http.StatusInternalServerError
			sysErr = fmt.Errorf("Parsing database response: %v", err)
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		}

		caps = append(caps, cap)
	}

	api.WriteResp(w, r, caps)
}

func Create(w http.ResponseWriter, r *http.Request) {
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	decoder := json.NewDecoder(r.Body)
	var cap tc.Capability
	if err := decoder.Decode(&cap); err != nil {
		sysErr = fmt.Errorf("Decoding request body: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	if cap.Name == "" {
		userErr = errors.New("'name' must be defined! (and not empty)")
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	if cap.Description == "" {
		userErr = errors.New("'description' must be defined! (and not empty)")
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	if ok, err := capabilityNameExists(cap.Name, tx); err != nil {
		sysErr = fmt.Errorf("Checking for capability %s's existence: %v", cap.Name, err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	} else if ok {
		userErr = fmt.Errorf("Capability '%s' already exists!", cap.Name)
		errCode = http.StatusConflict
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	row := tx.QueryRow(createQuery, cap.Name, cap.Description)
	if err := row.Scan(&cap.Description, &cap.LastUpdated, &cap.Name); err != nil {
		sysErr = fmt.Errorf("Inserting capability: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	alerts := tc.CreateAlerts(tc.SuccessLevel, "Capability created.")
	alerts.AddNewAlert(tc.WarnLevel, "This endpoint is deprecated, and will be removed in the future")

	api.WriteAlertsObj(w, r, http.StatusOK, alerts, cap)
	api.CreateChangeLogRawTx(api.ApiChange, fmt.Sprintf("CAPABILITY: %s, ACTION: Created", cap.Name), inf.User, tx)
}

func capabilityNameExists(c string, tx *sql.Tx) (bool, error) {
	row := tx.QueryRow(`SELECT name FROM capability WHERE name=$1`, c)
	var n string
	if err := row.Scan(&n); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
