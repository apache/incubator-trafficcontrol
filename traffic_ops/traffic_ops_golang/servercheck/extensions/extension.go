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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/jmoiron/sqlx"
)

// Create handler for creating a new servercheck extension.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	version := inf.Version.Major
	if inf.User.UserName != "extension" {
		errs.Code = http.StatusForbidden
		errs.SetUserError("invalid user for this API. Only the \"extension\" user can use this")
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
		return
	}

	toExt := tc.ServerCheckExtensionNullable{}

	// Validate request body
	if err := api.Parse(r.Body, inf.Tx.Tx, &toExt); err != nil {
		errs.Code = http.StatusBadRequest
		errs.UserError = err
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
		return
	}

	// Get Type ID
	typeID, exists, err := dbhelpers.GetTypeIDByName(*toExt.Type, inf.Tx.Tx)
	if !exists {
		errs.Code = http.StatusBadRequest
		errs.UserError = fmt.Errorf("type %v does not exist", *toExt.Type)
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
		return
	} else if err != nil {
		errs.Code = http.StatusInternalServerError
		errs.SystemError = err
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
		return
	}
	toExt.TypeID = &typeID

	successMsg := "Check Extension Loaded."
	errCode := http.StatusInternalServerError
	id, userErr, sysErr := createCheckExt(toExt, inf.Tx)
	// TODO: if a system error occurs - but not a user error - then the response code will be wrong (200 OK)
	if userErr != nil {
		errCode = http.StatusBadRequest
	}
	if userErr != nil || sysErr != nil {
		errs = apierrors.Errors{
			Code:        errCode,
			SystemError: sysErr,
			UserError:   userErr,
		}
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
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
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()
	version := inf.Version.Major

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"id":          dbhelpers.WhereColumnInfo{"e.id", api.IsInt},
		"name":        dbhelpers.WhereColumnInfo{"e.name", nil},
		"script_file": dbhelpers.WhereColumnInfo{"e.script_file", nil},
		"isactive":    dbhelpers.WhereColumnInfo{"e.isactive", api.IsBool},
		"type":        dbhelpers.WhereColumnInfo{"t.name", nil},
	}

	where, orderBy, pagination, queryValues, dbErrs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, queryParamsToSQLCols)
	if len(dbErrs) > 0 {
		errs.Code = http.StatusBadRequest
		errs.UserError = util.JoinErrs(dbErrs)
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
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
		errs.Code = http.StatusInternalServerError
		errs.SystemError = fmt.Errorf("querying to_extensions: %v", err)
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
		return
	}
	defer rows.Close()

	toExts := []tc.ServerCheckExtensionNullable{}
	for rows.Next() {
		toExt := tc.ServerCheckExtensionNullable{}
		if err = rows.StructScan(&toExt); err != nil {
			errs.Code = http.StatusInternalServerError
			errs.SystemError = fmt.Errorf("scanning to_extensions: %v", err)
			handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
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
	inf, errs := api.NewInfo(r, []string{"id"}, []string{"id"})
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()
	version := inf.Version.Major

	if inf.User.UserName != "extension" {
		errs.Code = http.StatusForbidden
		errs.SetUserError("invalid user for this API. Only the \"extension\" user can use this")
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
		return
	}

	id := inf.IntParams["id"]
	errs = deleteServerCheckExtension(id, inf.Tx)
	if errs.Occurred() {
		handleError(w, r, r.Method, inf.Tx.Tx, version, errs)
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

func deleteServerCheckExtension(id int, tx *sqlx.Tx) apierrors.Errors {
	// Get Open Slot Type ID
	errs := apierrors.New()
	openID, exists, err := dbhelpers.GetTypeIDByName("CHECK_EXTENSION_OPEN_SLOT", tx.Tx)
	if !exists {
		errs.SetSystemError("expected type CHECK_EXTENSION_OPEN_SLOT does not exist in type table")
		errs.Code = http.StatusInternalServerError
		return errs
	} else if err != nil {
		errs.SystemError = fmt.Errorf("getting CHECK_EXTENSION_OPEN_SLOT type id: %v", err)
		errs.Code = http.StatusInternalServerError
		return errs
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
		errs.SystemError = fmt.Errorf("deleting TO Extension: getting rows affected: %v", err)
		errs.Code = http.StatusInternalServerError
	} else if rowsAffected < 1 {
		errs.UserError = fmt.Errorf("no TO Extension with that key found")
		errs.Code = http.StatusNotFound
	} else if rowsAffected > 1 {
		errs.SystemError = fmt.Errorf("TO Extension delete affected too many rows: %d", rowsAffected)
		errs.Code = http.StatusInternalServerError
	}

	return errs
}

func handleError(w http.ResponseWriter, r *http.Request, httpMethod string, tx *sql.Tx, apiVersion uint64, errs apierrors.Errors) {
	if apiVersion < 2 {
		alt := "/servercheck/extensions"
		if httpMethod == http.MethodDelete {
			alt += "/:id"
		}
		api.HandleErrsOptionalDeprecation(w, r, tx, errs, true, util.StrPtr(fmt.Sprintf("%s %s", httpMethod, alt)))
	} else {
		api.HandleErrs(w, r, tx, errs)
	}
}
