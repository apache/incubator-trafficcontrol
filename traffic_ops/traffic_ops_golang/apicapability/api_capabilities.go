package apicapability

import (
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
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
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	results, err := getAPICapabilities(inf.Tx, inf.Params)
	if err.Occurred() {
		inf.HandleErrs(w, r, err)
		return
	}

	api.WriteResp(w, r, results)
	return
}

func getAPICapabilities(tx *sqlx.Tx, params map[string]string) ([]tc.APICapability, apierrors.Errors) {
	err := apierrors.New()
	selectQuery := `SELECT id, http_method, route, capability, last_updated FROM api_capability`
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"id":          dbhelpers.WhereColumnInfo{"id", api.IsInt},
		"capability":  dbhelpers.WhereColumnInfo{"capability", nil},
		"httpMethod":  dbhelpers.WhereColumnInfo{"http_method", nil},
		"route":       dbhelpers.WhereColumnInfo{"route", nil},
		"lastUpdated": dbhelpers.WhereColumnInfo{"last_updated", nil},
	}

	where, orderBy, pagination, queryValues, errs :=
		dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToQueryCols)

	if len(errs) > 0 {
		err.Code = http.StatusInternalServerError
		err.SystemError = fmt.Errorf("query exception: could not build api_capability query with params: %v, error: %v", params, util.JoinErrs(errs))
		return nil, err
	}

	query := selectQuery + where + orderBy + pagination
	rows, e := tx.NamedQuery(query, queryValues)

	if e != nil {
		return nil, api.ParseDBError(e)
	}
	defer rows.Close()

	apiCaps := []tc.APICapability{}
	for rows.Next() {
		var ac tc.APICapability
		e = rows.Scan(
			&ac.ID,
			&ac.HTTPMethod,
			&ac.Route,
			&ac.Capability,
			&ac.LastUpdated,
		)
		if e != nil {
			err.Code = http.StatusInternalServerError
			err.SystemError = fmt.Errorf("api capability read: scanning: %v", e)
			return nil, err
		}
		apiCaps = append(apiCaps, ac)
	}

	return apiCaps, err
}
