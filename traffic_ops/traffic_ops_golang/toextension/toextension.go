package toextension

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/jmoiron/sqlx"
)

// CreateTOExtension handler for creating a new TO Extension.
func CreateTOExtension(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	if inf.User.UserName != "extension" {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusForbidden, errors.New("invalid user for this API. Only the \"extension\" user can use this"), nil)
		return
	}

	toExt := tc.TOExtensionNullable{}

	// Validate request body
	if err := api.Parse(r.Body, inf.Tx.Tx, &toExt); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	// Get Type ID
	typeID, exists, err := dbhelpers.GetTypeIDByName(*toExt.Type, inf.Tx.Tx)
	if !exists {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, fmt.Errorf("type %v does not exist", *toExt.Type), nil)
		return
	} else if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
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
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	resp := tc.TOExtensionPostResponse{
		Response: tc.TOExtensionID{ID: id},
		Alerts:   tc.CreateAlerts(tc.SuccessLevel, successMsg),
	}
	changeLogMsg := fmt.Sprintf("TO_EXTENSION: %s, ID: %d, ACTION: CREATED", *toExt.Name, id)

	api.CreateChangeLogRawTx(api.ApiChange, changeLogMsg, inf.User, inf.Tx.Tx)

	api.WriteRespRaw(w, r, resp)
}

func createCheckExt(toExt tc.TOExtensionNullable, tx *sqlx.Tx) (int, error, error) {
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
