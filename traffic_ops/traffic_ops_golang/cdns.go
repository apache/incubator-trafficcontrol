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

const CDNsPrivLevel = 10

func cdnsHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErr := func(err error, status int) {
			log.Errorf("%v %v\n", r.RemoteAddr, err)
			w.WriteHeader(status)
			fmt.Fprintf(w, http.StatusText(status))
		}

		q := r.URL.Query()

		resp, err := getCDNsResponse(q, db)

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

func getCDNsResponse(q url.Values, db *sqlx.DB) (*tc.CDNsResponse, error) {
	CDNs, err := getCDNs(q, db)
	if err != nil {
		return nil, fmt.Errorf("getting CDNs response: %v", err)
	}

	resp := tc.CDNsResponse{
		Response: CDNs,
	}
	return &resp, nil
}

func getCDNs(v url.Values, db *sqlx.DB) ([]tc.CDN, error) {
	var rows *sqlx.Rows
	var err error

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToQueryCols := map[string]string{
		"domainName":    "domain_name",
		"dnssecEnabled": "dnssec_enabled",
		"id":            "id",
		"name":          "name",
	}

	query, queryValues := BuildQuery(v, selectCDNsQuery(), queryParamsToQueryCols)

	rows, err = db.NamedQuery(query, queryValues)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	CDNs := []tc.CDN{}
	for rows.Next() {
		var s tc.CDN
		if err = rows.StructScan(&s); err != nil {
			return nil, fmt.Errorf("getting CDNs: %v", err)
		}
		CDNs = append(CDNs, s)
	}
	return CDNs, nil
}

func selectCDNsQuery() string {

	query := `SELECT
dnssec_enabled,
domain_name,
id,
last_updated,
name 

FROM cdn c`
	return query
}
