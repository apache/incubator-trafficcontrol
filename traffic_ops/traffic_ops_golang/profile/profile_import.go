package profile

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
	"github.com/lib/pq"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

// ImportProfileHandler handles importing profile
func ImportProfileHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	importedProfile := tc.ProfileImportRequest{}

	if err := api.Parse(r.Body, inf.Tx.Tx, &importedProfile); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	id, err := importProfile(&importedProfile.Profile, inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("importing profile: "+err.Error()))
		return
	}

	importedProfileResponse := tc.ProfileImportResponseObj{
		ProfileExportImportNullable: importedProfile.Profile,
		ID:                          &id,
	}

	newParamCnt, existingParamCnt, err := importProfileParameters(id, importedProfile.Parameters, inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("importing profile parameters: "+err.Error()))
		return
	}

	successMsg := fmt.Sprintf("Profile imported [ %v ] with %v new and %v existing parameters",
		*importedProfile.Profile.Name, newParamCnt, existingParamCnt)

	api.CreateChangeLogRawTx(api.ApiChange, successMsg, inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, successMsg, importedProfileResponse)
}

func importProfile(importedProfile *tc.ProfileExportImportNullable, tx *sql.Tx) (int, error) {
	var id int
	insertQuery := `
INSERT INTO profile (name, description, cdn, type)
SELECT $1, $2, id, $4
FROM cdn
WHERE name = $3
RETURNING id`
	if err := tx.QueryRow(
		insertQuery,
		importedProfile.Name,
		importedProfile.Description,
		importedProfile.CDNName,
		importedProfile.Type).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, fmt.Errorf("imported profile %v was not inserted, no id was returned", importedProfile.Name)
		}
		return id, errors.New("scanning profile id after insert: " + err.Error())
	}
	return id, nil
}

func importProfileParameters(profileID int, importedParameters []tc.ProfileExportImportParameterNullable, tx *sql.Tx) (int, int, error) {
	if len(importedParameters) == 0 {
		return 0, 0, nil
	}
	idSet := map[int]struct{}{}
	ids := []int{}
	existingCnt := 0
	newCnt := 0
	selectQuery := `
SELECT id
FROM parameter
WHERE name=$1 AND config_file=$2 AND value=$3`

	for _, param := range importedParameters {
		var id int
		existingParam := true
		if err := tx.QueryRow(selectQuery, param.Name, param.ConfigFile, param.Value).Scan(&id); err != nil {
			if err == sql.ErrNoRows {
				existingParam = false
			} else {
				return 0, 0, errors.New("querying parameter: " + err.Error())
			}
		}
		if existingParam {
			if _, ok := idSet[id]; ok {
				continue
			}
			ids = append(ids, id)
			idSet[id] = struct{}{}
			existingCnt++
		} else {
			// Insert Parameter
			newID, err := insertParameter(&param, tx)
			if err != nil {
				return 0, 0, err
			}
			ids = append(ids, newID)
			idSet[newID] = struct{}{}
			newCnt++
		}
	}

	// Insert profile, parameter records
	insertPPQuery := `
INSERT INTO profile_parameter (profile, parameter)
VALUES ($1, unnest($2::int[]))`
	if _, err := tx.Exec(insertPPQuery, profileID, pq.Array(ids)); err != nil {
		return 0, 0, errors.New("inserting profile parameter: " + err.Error())
	}
	return newCnt, existingCnt, nil
}

func insertParameter(param *tc.ProfileExportImportParameterNullable, tx *sql.Tx) (int, error) {
	var id int
	insertQuery := `
INSERT INTO parameter (
name,
config_file,
value) VALUES ($1,$2,$3) RETURNING id`
	if err := tx.QueryRow(insertQuery, param.Name, param.ConfigFile, param.Value).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, fmt.Errorf("imported parameter %v was not inserted, no id was returned", param.Name)
		}
		return id, errors.New("scanning parameter id after insert: " + err.Error())
	}
	return id, nil
}
