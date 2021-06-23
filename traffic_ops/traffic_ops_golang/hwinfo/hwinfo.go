package hwinfo

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
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/jmoiron/sqlx"
)

const selectHWInfoQuery = `
SELECT
	s.host_name as serverhostname,
	h.id,
	h.serverid,
	h.description,
	h.val,
	h.last_updated
FROM hwinfo h
JOIN server s ON s.id = h.serverid
`

// Get handles GET requests to /hwinfo
func Get(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	alerts := tc.CreateAlerts(tc.WarnLevel, "This endpoint is deprecated, and will be removed in the future")

	// Mimic Perl behavior
	if _, ok := inf.Params["limit"]; !ok {
		inf.Params["limit"] = "1000"
	}
	limit, err := strconv.ParseUint(inf.Params["limit"], 10, 64)
	if err != nil || limit == 0 {
		alerts.AddNewAlert(tc.ErrorLevel, "'limit' parameter must be a positive integer")
		api.WriteAlerts(w, r, http.StatusBadRequest, alerts)
		return
	}

	hwInfo, err := getHWInfo(tx, inf.Params)
	if err != nil {
		log.Errorln(err.Error())
		alerts.AddNewAlert(tc.ErrorLevel, http.StatusText(http.StatusInternalServerError))
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}

	resp := struct {
		tc.Alerts
		Response []tc.HWInfo `json:"response"`
		Limit    uint64      `json:"limit"`
	}{
		Alerts:   alerts,
		Response: hwInfo,
		Limit:    limit,
	}

	var respBts []byte
	if respBts, err = json.Marshal(resp); err != nil {
		log.Errorf("Marshaling JSON: %v", err)
		alerts.AddNewAlert(tc.ErrorLevel, http.StatusText(http.StatusInternalServerError))
		api.WriteAlerts(w, r, http.StatusInternalServerError, alerts)
		return
	}

	w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
	w.Write(append(respBts, '\n'))
}

func getHWInfo(tx *sqlx.Tx, params map[string]string) ([]tc.HWInfo, error) {

	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"id":             dbhelpers.WhereColumnInfo{Column: "h.id", Checker: api.IsInt},
		"serverHostName": dbhelpers.WhereColumnInfo{Column: "s.host_name"},
		"serverId":       dbhelpers.WhereColumnInfo{Column: "s.id", Checker: api.IsInt},
		"description":    dbhelpers.WhereColumnInfo{Column: "h.description"},
		"val":            dbhelpers.WhereColumnInfo{Column: "h.val"},
		"lastUpdated":    dbhelpers.WhereColumnInfo{Column: "h.last_updated"}, //TODO: this doesn't appear to work needs debugging
	}
	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, fmt.Errorf("Building hwinfo query clauses: %v", util.JoinErrs(errs))
	}

	rows, err := tx.NamedQuery(selectHWInfoQuery+where+orderBy+pagination, queryValues)
	if err != nil {
		return nil, fmt.Errorf("querying hwinfo: %v", err)
	}
	defer rows.Close()

	hwInfo := []tc.HWInfo{}
	for rows.Next() {
		var info tc.HWInfo
		if err = rows.StructScan(&info); err != nil {
			return nil, fmt.Errorf("scanning hwinfo: %v", err)
		}

		hwInfo = append(hwInfo, info)
	}

	return hwInfo, nil
}
