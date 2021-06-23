package extensions

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
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/jmoiron/sqlx"
)

// Create handler for creating a new servercheck extension.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	version := inf.Version.Major
	if inf.User.UserName != "extension" {
		handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusForbidden, errors.New("invalid user for this API. Only the \"extension\" user can use this"), nil)
		return
	}

	toExt := tc.ServerCheckExtensionNullable{}

	// Validate request body
	if err := api.Parse(r.Body, inf.Tx.Tx, &toExt); err != nil {
		handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusBadRequest, err, nil)
		return
	}

	// Get Type ID
	typeID, exists, err := dbhelpers.GetTypeIDByName(*toExt.Type, inf.Tx.Tx)
	if !exists {
		handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusBadRequest, fmt.Errorf("type %v does not exist", *toExt.Type), nil)
		return
	} else if err != nil {
		handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusInternalServerError, nil, err)
		return
	}
	toExt.TypeID = &typeID

	successMsg := "Check Extension Loaded."
	errCode = http.StatusInternalServerError
	id, userErr, sysErr := createCheckExt(toExt, inf.Tx)
	if userErr != nil {
		errCode = http.StatusBadRequest
	}
	if userErr != nil || sysErr != nil {
		handleError(w, r, r.Method, inf.Tx.Tx, version, errCode, userErr, sysErr)
		return
	}
	resp := tc.ServerCheckExtensionPostResponse{
		Response: tc.ServerCheckExtensionID{ID: id},
		Alerts:   tc.CreateAlerts(tc.SuccessLevel, successMsg),
	}

	if version < 2 {
		resp.AddNewAlert(tc.WarnLevel, "This endpoint is deprecated, please use POST /servercheck/extensions instead")
	}

	changeLogMsg := fmt.Sprintf("TO_EXTENSION: %s, ID: %d, ACTION: CREATED", *toExt.Name, id)

	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, inf.Tx.Tx)

	api.WriteRespRaw(w, r, resp)
}

func createCheckExt(toExt tc.ServerCheckExtensionNullable, tx *sqlx.Tx) (int, error, error) {
	id := 0
	dupErr, sysErr := checkDupTOCheckExtension("name", *toExt.Name, tx)
	if dupErr != nil || sysErr != nil {
		return 0, dupErr, sysErr
	}

	dupErr, sysErr = checkDupTOCheckExtension("servercheck_short_name", *toExt.ServercheckShortName, tx)
	if dupErr != nil || sysErr != nil {
		return 0, dupErr, sysErr
	}

	// Get open slot
	scc := ""
	if err := tx.Tx.QueryRow(`
	SELECT id, servercheck_column_name
	FROM to_extension
	WHERE type in
		(SELECT id FROM type WHERE name = 'CHECK_EXTENSION_OPEN_SLOT')
	ORDER BY servercheck_column_name
	LIMIT 1`).Scan(&id, &scc); err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("No open slots left for checks, delete one first."), nil

		}
		return 0, nil, fmt.Errorf("querying open slot to_extension: %v", err)
	}
	toExt.ID = &id
	_, err := tx.NamedExec(updateQuery(), toExt)
	if err != nil {
		return 0, nil, fmt.Errorf("update open extension slot to new check extension: %v", err)
	}

	_, err = tx.Tx.Exec(fmt.Sprintf("UPDATE servercheck set %v = 0", scc))
	if err != nil {
		return 0, nil, fmt.Errorf("reset servercheck table for new check extension: %v", err)
	}
	return id, nil, nil
}

func checkDupTOCheckExtension(colName, value string, tx *sqlx.Tx) (error, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT * FROM to_extension WHERE %v =$1)", colName)
	exists := false
	err := tx.Tx.QueryRow(query, value).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking if to_extension %v already exists: %v", colName, err)
	}
	if exists {
		return fmt.Errorf("A Check extension is already loaded with %v %v", value, colName), nil
	}
	return nil, nil
}

func updateQuery() string {
	return `
	UPDATE to_extension SET
	name=:name,
	version=:version,
	info_url=:info_url,
	script_file=:script_file,
	isactive=:isactive,
	additional_config_json=:additional_config_json,
	description=:description,
	servercheck_short_name=:servercheck_short_name,
	type=:type
	WHERE id=:id
	`
}

func selectQuery() string {
	return `
	SELECT
		e.id,
		e.name,
		e.version,
		e.info_url,
		e.script_file,
		e.isactive::::int,
		e.additional_config_json,
		e.description,
		e.servercheck_short_name,
		t.name AS type_name
	FROM to_extension AS e
	JOIN type AS t ON e.type = t.id
	`
}

// Get handler for getting servercheck extensions.
func Get(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	version := inf.Version.Major

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"id":          dbhelpers.WhereColumnInfo{Column: "e.id", Checker: api.IsInt},
		"name":        dbhelpers.WhereColumnInfo{Column: "e.name"},
		"script_file": dbhelpers.WhereColumnInfo{Column: "e.script_file"},
		"isactive":    dbhelpers.WhereColumnInfo{Column: "e.isactive", Checker: api.IsBool},
		"type":        dbhelpers.WhereColumnInfo{Column: "t.name"},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, queryParamsToSQLCols)
	if len(errs) > 0 {
		handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusBadRequest, util.JoinErrs(errs), nil)
		return
	}

	openSlotCond := "t.name != 'CHECK_EXTENSION_OPEN_SLOT'"
	if len(where) > 0 {
		where = fmt.Sprintf("%s AND %s", where, openSlotCond)
	} else {
		where = fmt.Sprintf("%s %s", dbhelpers.BaseWhere, openSlotCond)
	}

	query := selectQuery() + where + orderBy + pagination
	log.Infoln(query)

	rows, err := inf.Tx.NamedQuery(query, queryValues)
	if err != nil {
		handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusInternalServerError, nil, fmt.Errorf("querying to_extensions: %v", err))
		return
	}
	defer rows.Close()

	toExts := []tc.ServerCheckExtensionNullable{}
	for rows.Next() {
		toExt := tc.ServerCheckExtensionNullable{}
		if err = rows.StructScan(&toExt); err != nil {
			handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusInternalServerError, nil, fmt.Errorf("scanning to_extensions: %v", err))
			return
		}
		toExts = append(toExts, toExt)
	}
	if version < 2 {
		api.WriteRespAlertObj(w, r, tc.WarnLevel, "This endpoint is deprecated, please use GET /servercheck/extensions instead", toExts)
	} else {
		api.WriteResp(w, r, toExts)
	}
}

// Delete is the handler for deleting servercheck extensions.
func Delete(w http.ResponseWriter, r *http.Request) {
	inf, sysErr, userErr, errCode := api.NewInfo(w, r, []string{"id"}, []string{"id"})
	tx := inf.Tx.Tx
	if sysErr != nil || userErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	version := inf.Version.Major

	if inf.User.UserName != "extension" {
		handleError(w, r, r.Method, inf.Tx.Tx, version, http.StatusForbidden, errors.New("invalid user for this API. Only the \"extension\" user can use this"), nil)
		return
	}

	id := inf.IntParams["id"]
	userErr, sysErr, errCode = deleteServerCheckExtension(id, inf.Tx)
	if userErr != nil || sysErr != nil {
		handleError(w, r, r.Method, inf.Tx.Tx, version, errCode, userErr, sysErr)
		return
	}

	changeLogMsg := fmt.Sprintf("TO_EXTENSION: %d, ID: %d, ACTION: Deleted", id, id)
	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, inf.Tx.Tx)
	alerts := tc.CreateAlerts(tc.SuccessLevel, "Extensions deleted.")
	if version < 2 {
		alerts.AddNewAlert(tc.WarnLevel, "This endpoint is deprecated, please use DELETE /servercheck/extensions/:id instead")
	}
	api.WriteAlerts(w, r, http.StatusOK, alerts)
}

func deleteServerCheckExtension(id int, tx *sqlx.Tx) (error, error, int) {
	// Get Open Slot Type ID
	openID, exists, err := dbhelpers.GetTypeIDByName("CHECK_EXTENSION_OPEN_SLOT", tx.Tx)
	if !exists {
		return nil, errors.New("expected type CHECK_EXTENSION_OPEN_SLOT does not exist in type table"), http.StatusInternalServerError
	} else if err != nil {
		return nil, fmt.Errorf("getting CHECK_EXTENSION_OPEN_SLOT type id: %v", err), http.StatusInternalServerError
	}

	openTOExt := tc.ServerCheckExtensionNullable{
		Name:                 util.StrPtr("OPEN"),
		Version:              util.StrPtr("0"),
		InfoURL:              util.StrPtr(""),
		ScriptFile:           util.StrPtr(""),
		IsActive:             util.IntPtr(0),
		AdditionConfigJSON:   util.StrPtr(""),
		ServercheckShortName: util.StrPtr(""),
		TypeID:               &openID,
		ID:                   &id,
	}

	result, err := tx.NamedExec(updateQuery(), openTOExt)
	if err != nil {
		return api.ParseDBError(err)
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return nil, fmt.Errorf("deleting TO Extension: getting rows affected: %v", err), http.StatusInternalServerError
	} else if rowsAffected < 1 {
		return fmt.Errorf("no TO Extension with that key found"), nil, http.StatusNotFound
	} else if rowsAffected > 1 {
		return nil, fmt.Errorf("TO Extension delete affected too many rows: %d", rowsAffected), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

func handleError(w http.ResponseWriter, r *http.Request, httpMethod string, tx *sql.Tx, apiVersion uint64, errCode int, userErr, sysErr error) {
	if apiVersion < 2 {
		alt := "/servercheck/extensions"
		if httpMethod == http.MethodDelete {
			alt += "/:id"
		}
		api.HandleDeprecatedErr(w, r, tx, errCode, userErr, sysErr, util.StrPtr(fmt.Sprintf("%s %s", httpMethod, alt)))
	} else {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
	}
}
