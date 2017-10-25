package main

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
	"fmt"
	"net/http"
	"net/url"

	"github.com/apache/incubator-trafficcontrol/lib/go-log"
	tc "github.com/apache/incubator-trafficcontrol/lib/go-tc"
	"github.com/jmoiron/sqlx"
)

const HWInfoPrivLevel = 10

func hwInfoHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErr := func(err error, status int) {
			log.Errorf("%v %v\n", r.RemoteAddr, err)
			w.WriteHeader(status)
			fmt.Fprintf(w, http.StatusText(status))
		}

		q := r.URL.Query()
		resp, err := getHWInfoResponse(q, db)
		if err != nil {
			handleErr(err, http.StatusInternalServerError)
			return
		}

		respBts, err := json.Marshal(resp)
		if err != nil {
			handleErr(err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", respBts)
	}
}

func getHWInfoResponse(q url.Values, db *sqlx.DB) (*tc.HWInfoResponse, error) {
	hwInfo, err := getHWInfo(q, db)
	if err != nil {
		return nil, fmt.Errorf("getting hwInfo response: %v", err)
	}

	resp := tc.HWInfoResponse{
		Response: hwInfo,
	}
	return &resp, nil
}

func getHWInfo(v url.Values, db *sqlx.DB) ([]tc.HWInfo, error) {
	var rows *sqlx.Rows
	var err error

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]string{
		"id":             "h.id",
		"serverHostName": "s.serverHostName",
		"serverId":       "s.serverid",
		"description":    "h.description",
		"val":            "h.val",
		"lastUpdated":    "h.last_updated",
	}

	query, queryValues := BuildQuery(v, selectHWInfoQuery(), queryParamsToSQLCols)

	rows, err = db.NamedQuery(query, queryValues)

	if err != nil {
		return nil, err
	}
	hwInfo := []tc.HWInfo{}

	defer rows.Close()
	for rows.Next() {
		var s tc.HWInfo
		if err = rows.StructScan(&s); err != nil {
			return nil, fmt.Errorf("getting hwInfo: %v", err)
		}
		hwInfo = append(hwInfo, s)
	}
	return hwInfo, nil
}

func selectHWInfoQuery() string {

	query := `SELECT
	s.host_name as serverhostname,
    h.id,
    h.serverid,
    h.description,
    h.val,
    h.last_updated

FROM hwInfo h

JOIN server s ON s.id = h.serverid`
	return query
}
