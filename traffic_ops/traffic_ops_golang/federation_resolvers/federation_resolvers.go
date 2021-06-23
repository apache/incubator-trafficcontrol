// Package federation_resolvers contains handler logic for the /federation_resolvers and
// /federation_resolvers/{{ID}} endpoints.
package federation_resolvers

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
	"fmt"
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/jmoiron/sqlx"
)

const insertFederationResolverQuery = `
INSERT INTO federation_resolver (ip_address, type)
VALUES ($1, $2)
RETURNING federation_resolver.id,
          federation_resolver.ip_address,
          (
          	SELECT type.name
          	FROM type
          	WHERE type.id = federation_resolver.type
          ) AS type,
          federation_resolver.type as typeId
`

const readQuery = `
SELECT federation_resolver.id,
       federation_resolver.ip_address,
       federation_resolver.last_updated,
       type.name AS type
FROM federation_resolver
LEFT OUTER JOIN type ON type.id = federation_resolver.type
`

const deleteQuery = `
DELETE FROM federation_resolver
WHERE federation_resolver.id = $1
RETURNING federation_resolver.id,
          federation_resolver.ip_address,
          (
          	SELECT type.name
          	FROM type
          	WHERE type.id = federation_resolver.type
          ) AS type
`

// Create is the handler for POST requests to /federation_resolvers.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if sysErr != nil || userErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var fr tc.FederationResolver
	if userErr = api.Parse(r.Body, tx, &fr); userErr != nil {
		api.HandleErr(w, r, tx, http.StatusBadRequest, userErr, nil)
		return
	}

	err := tx.QueryRow(insertFederationResolverQuery, fr.IPAddress, fr.TypeID).Scan(&fr.ID, &fr.IPAddress, &fr.Type, &fr.TypeID)
	if err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if inf.Version.Major == 1 && inf.Version.Minor < 4 {
		fr.LastUpdated = nil
		fr.Type = nil
	}

	changeLogMsg := fmt.Sprintf("FEDERATION_RESOLVER: %s, ID: %d, ACTION: Created", *fr.IPAddress, *fr.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, tx)

	alertMsg := fmt.Sprintf("Federation Resolver created [ IP = %s ] with id: %d", *fr.IPAddress, *fr.ID)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, alertMsg, fr)
}

// Read is the handler for GET requests to /federation_resolvers (and /federation_resolvers/{{ID}}).
func Read(w http.ResponseWriter, r *http.Request) {
	var maxTime time.Time
	var runSecond bool
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, nil, nil)
	tx := inf.Tx.Tx
	if sysErr != nil || userErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"id":        dbhelpers.WhereColumnInfo{Column: "federation_resolver.id", Checker: api.IsInt},
		"ipAddress": dbhelpers.WhereColumnInfo{Column: "federation_resolver.ip_address"},
		"type":      dbhelpers.WhereColumnInfo{Column: "type.name"},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, queryParamsToQueryCols)
	if len(errs) > 0 {
		sysErr = util.JoinErrs(errs)
		errCode = http.StatusBadRequest
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	query := readQuery + where + orderBy + pagination
	useIMS := false
	config, e := api.GetConfig(r.Context())
	if e == nil && config != nil {
		useIMS = config.UseIMS
	} else {
		log.Warnf("Couldn't get config %v", e)
	}
	if useIMS {
		runSecond, maxTime = TryIfModifiedSinceQuery(r.Header, inf.Tx, where, orderBy, pagination, queryValues)
		if !runSecond {
			log.Debugln("IMS HIT")
			api.AddLastModifiedHdr(w, maxTime)
			w.WriteHeader(http.StatusNotModified)
			api.WriteResp(w, r, []tc.FederationResolver{})
			return
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	rows, err := inf.Tx.NamedQuery(query, queryValues)
	if err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		if sysErr != nil {
			sysErr = fmt.Errorf("federation_resolver read query: %v", sysErr)
		}

		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer rows.Close()

	var resolvers = []tc.FederationResolver{}
	for rows.Next() {
		var resolver tc.FederationResolver
		if err := rows.Scan(&resolver.ID, &resolver.IPAddress, &resolver.LastUpdated, &resolver.Type); err != nil {
			userErr, sysErr, errCode = api.ParseDBError(err)
			if sysErr != nil {
				sysErr = fmt.Errorf("federation_resolver scanning: %v", sysErr)
			}
			api.HandleErr(w, r, tx, errCode, userErr, sysErr)
			return
		}

		resolvers = append(resolvers, resolver)
	}

	if api.SetLastModifiedHeader(r, useIMS) {
		// RFC1123
		date := maxTime.Format("Mon, 02 Jan 2006 15:04:05 MST")
		w.Header().Add(rfc.LastModified, date)
	}
	api.WriteResp(w, r, resolvers)
}

func TryIfModifiedSinceQuery(header http.Header, tx *sqlx.Tx, where string, orderBy string, pagination string, queryValues map[string]interface{}) (bool, time.Time) {
	var max time.Time
	var imsDate time.Time
	var ok bool
	imsDateHeader := []string{}
	runSecond := true
	dontRunSecond := false
	if header == nil {
		return runSecond, max
	}
	imsDateHeader = header[rfc.IfModifiedSince]
	if len(imsDateHeader) == 0 {
		return runSecond, max
	}
	if imsDate, ok = rfc.ParseHTTPDate(imsDateHeader[0]); !ok {
		log.Warnf("IMS request header date '%s' not parsable", imsDateHeader[0])
		return runSecond, max
	}
	query := SelectMaxLastUpdatedQuery(where, "federation_resolver")
	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		log.Warnf("Couldn't get the max last updated time: %v", err)
		return runSecond, max
	}
	if err == sql.ErrNoRows {
		return dontRunSecond, max
	}
	defer rows.Close()
	// This should only ever contain one row
	if rows.Next() {
		v := &ims.LatestTimestamp{}
		if err = rows.StructScan(v); err != nil || v == nil {
			log.Warnf("Failed to parse the max time stamp into a struct %v", err)
			return runSecond, max
		}
		if v.LatestTime != nil {
			max = v.LatestTime.Time
			// The request IMS time is later than the max of (lastUpdated, deleted_time)
			if imsDate.After(v.LatestTime.Time) {
				return dontRunSecond, max
			}
		}
	}
	return runSecond, max
}

func SelectMaxLastUpdatedQuery(where string, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}

// Delete is the handler for DELETE requests to /federation_resolvers.
func Delete(w http.ResponseWriter, r *http.Request) {
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if sysErr != nil || userErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	alert, respObj, userErr, sysErr, statusCode := deleteFederationResolver(inf)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, statusCode, userErr, sysErr)
		return
	}

	api.WriteRespAlertObj(w, r, tc.SuccessLevel, alert.Text, respObj)
}

func deleteFederationResolver(inf *api.Info) (tc.Alert, tc.FederationResolver, error, error, int) {
	var userErr error
	var sysErr error
	var statusCode = http.StatusOK
	var alert tc.Alert
	var result tc.FederationResolver

	err := inf.Tx.Tx.QueryRow(deleteQuery, inf.IntParams["id"]).Scan(&result.ID, &result.IPAddress, &result.Type)
	if err != nil {
		if err == sql.ErrNoRows {
			userErr = fmt.Errorf("No federation resolver by ID %d", inf.IntParams["id"])
			statusCode = http.StatusNotFound
		} else {
			userErr, sysErr, statusCode = api.ParseDBError(err)
		}

		return alert, result, userErr, sysErr, statusCode
	}

	changeLogMsg := fmt.Sprintf("FEDERATION_RESOLVER: %s, ID: %d, ACTION: Deleted", *result.IPAddress, *result.ID)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, inf.Tx.Tx)

	alertMsg := fmt.Sprintf("Federation resolver deleted [ IP = %s ] with id: %d", *result.IPAddress, *result.ID)
	alert = tc.Alert{
		Level: tc.SuccessLevel.String(),
		Text:  alertMsg,
	}

	return alert, result, userErr, sysErr, statusCode
}

// DeleteByID is the handler for DELETE requests to /federation_resolvers/{{ID}}.
func DeleteByID(w http.ResponseWriter, r *http.Request) {
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if sysErr != nil || userErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	alert, respObj, userErr, sysErr, statusCode := deleteFederationResolver(inf)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, statusCode, userErr, sysErr)
		return
	}

	var resp = struct {
		tc.Alerts
		Response *tc.FederationResolver `json:"response,omitempty"`
	}{
		tc.Alerts{
			Alerts: []tc.Alert{
				alert,
				tc.Alert{
					Level: tc.WarnLevel.String(),
					Text:  "This endpoint is deprecated, please use the 'id' query parameter of '/federation_resolvers' instead",
				},
			},
		},
		&respObj,
	}

	if inf.Version.Major == 1 && inf.Version.Minor < 5 {
		resp.Response = nil
	}

	respBts, err := json.Marshal(resp)
	if err != nil {
		sysErr = fmt.Errorf("marhsaling response: %v", err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, statusCode, userErr, sysErr)
		return
	}

	w.Header().Set(rfc.ContentType, rfc.MIME_JSON.String())
	w.WriteHeader(statusCode)
	w.Write(append(respBts, '\n'))
}
