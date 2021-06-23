package profileparameter

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
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/lib/pq"
)

func PostProfileParamsByName(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	profParams := tc.ProfileParametersByNamePost{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &profParams); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parse error: "+err.Error()), nil)
		return
	}
	profileName := inf.Params["name"]
	profileID, ok, err := dbhelpers.GetProfileIDFromName(profileName, inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting profile '"+profileName+"' ID: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("no profile with that name exists"), nil)
		return
	}
	insertedObjs, err := insertParametersForProfile(profileName, profParams, inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("posting profile parameters by name: "+err.Error()))
		return
	}
	resp := tc.ProfileParameterPostResp{Parameters: insertedObjs, ProfileName: profileName, ProfileID: profileID}
	api.CreateChangeLogRawTx(api.ApiChange, "PROFILE: "+profileName+", ID: "+strconv.Itoa(profileID)+", ACTION: Assigned parameters to profile", inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Assign parameters successfully to profile "+profileName, resp)
}

// insertParametersForProfile returns the PostResp object, because the ID is needed, and the ID must be associated with the real key (name,value,config_file), so we might as well return the whole object.
func insertParametersForProfile(profileName string, params tc.ProfileParametersByNamePost, tx *sql.Tx) ([]tc.ProfileParameterPostRespObj, error) {
	insertParamsQ := `
INSERT INTO parameter (name, config_file, value, secure)
VALUES (unnest($1::text[]), unnest($2::text[]), unnest($3::text[]), unnest($4::bool[]))
ON CONFLICT(name, config_file, value) DO UPDATE set name=EXCLUDED.name RETURNING id, name, config_file, value, secure;
`
	paramNames := make([]string, len(params))
	paramConfigFiles := make([]string, len(params))
	paramValues := make([]string, len(params))
	paramSecures := make([]bool, len(params))
	for i, param := range params {
		paramNames[i] = *param.Name
		paramConfigFiles[i] = *param.ConfigFile
		paramValues[i] = *param.Value
		if *param.Secure != 0 {
			paramSecures[i] = true
		}
	}
	rows, err := tx.Query(insertParamsQ, pq.Array(paramNames), pq.Array(paramConfigFiles), pq.Array(paramValues), pq.Array(paramSecures))
	if err != nil {
		return nil, errors.New("querying post parameters for profile: " + err.Error())
	}
	defer rows.Close()
	ids := make([]int64, 0, len(params))
	insertedObjs := []tc.ProfileParameterPostRespObj{}
	for rows.Next() {
		id := int64(0)
		name := ""
		configFile := ""
		value := ""
		secure := false
		secureNum := 0
		if err := rows.Scan(&id, &name, &configFile, &value, &secure); err != nil {
			return nil, errors.New("scanning new parameter IDs: " + err.Error())
		}
		if secure {
			secureNum = 1
		}
		ids = append(ids, id)
		insertedObjs = append(insertedObjs, tc.ProfileParameterPostRespObj{ID: id, ProfileParameterByNamePost: tc.ProfileParameterByNamePost{Name: &name, ConfigFile: &configFile, Value: &value, Secure: &secureNum}})
	}
	insertProfileParamsQ := `
INSERT INTO profile_parameter (profile, parameter)
VALUES ((SELECT id FROM profile WHERE name = $1), unnest($2::int[]))
ON CONFLICT DO NOTHING;
`
	if _, err := tx.Exec(insertProfileParamsQ, profileName, pq.Array(ids)); err != nil {
		return nil, errors.New("inserting profile parameters: " + err.Error())
	}

	return insertedObjs, nil
}
