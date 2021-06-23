package apicapability

import (
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/jmoiron/sqlx"
)

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

// GetAPICapabilitiesHandler implements an http handler that returns
// API Capabilities. In the event a capability parameter is supplied,
// it will return only those with an exact match.
func GetAPICapabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	results, errCode, usrErr, sysErr := getAPICapabilities(inf.Tx, inf.Params)
	if usrErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, usrErr, sysErr)
		return
	}

	api.WriteResp(w, r, results)
	return
}

func getAPICapabilities(tx *sqlx.Tx, params map[string]string) ([]tc.APICapability, int, error, error) {
	var err error
	selectQuery := `SELECT id, http_method, route, capability, last_updated FROM api_capability`
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"id":          dbhelpers.WhereColumnInfo{Column: "id", Checker: api.IsInt},
		"capability":  dbhelpers.WhereColumnInfo{Column: "capability"},
		"httpMethod":  dbhelpers.WhereColumnInfo{Column: "http_method"},
		"route":       dbhelpers.WhereColumnInfo{Column: "route"},
		"lastUpdated": dbhelpers.WhereColumnInfo{Column: "last_updated"},
	}

	where, orderBy, pagination, queryValues, errs :=
		dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToQueryCols)

	if len(errs) > 0 {
		err = util.JoinErrs(errs)
		return nil, http.StatusInternalServerError, nil, fmt.Errorf(
			"query exception: could not build api_capability query with params: %v, error: %v",
			params,
			err,
		)
	}

	query := selectQuery + where + orderBy + pagination
	rows, err := tx.NamedQuery(query, queryValues)

	if err != nil {
		usrErr, sysErr, errCode := api.ParseDBError(err)
		return nil, errCode, usrErr, sysErr
	}
	defer rows.Close()

	apiCaps := []tc.APICapability{}
	for rows.Next() {
		var ac tc.APICapability
		err = rows.Scan(
			&ac.ID,
			&ac.HTTPMethod,
			&ac.Route,
			&ac.Capability,
			&ac.LastUpdated,
		)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, fmt.Errorf(
				"api capability read: scanning: %v", err,
			)
		}
		apiCaps = append(apiCaps, ac)
	}

	return apiCaps, 0, nil, nil
}
