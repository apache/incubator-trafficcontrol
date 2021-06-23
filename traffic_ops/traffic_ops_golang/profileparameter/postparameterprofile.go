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
	"fmt"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"

	"github.com/lib/pq"
)

func PostParamProfile(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	paramProfile := tc.PostParamProfile{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &paramProfile); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parse error: "+err.Error()), nil)
		return
	}
	if err := insertParameterProfile(paramProfile, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("posting parameter profile: "+err.Error()))
		return
	}
	paramName, ok, err := getParamNameFromID(*paramProfile.ParamID, inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting parameter name from id: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("parameter not found"), nil)
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "PARAM: "+paramName+", ID: "+strconv.FormatInt(*paramProfile.ParamID, 10)+", ACTION: Assigned "+strconv.Itoa(len(*paramProfile.ProfileIDs))+" profiles to parameter", inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, fmt.Sprintf("%d profiles were assigned to the %d parameter", len(*paramProfile.ProfileIDs), *paramProfile.ParamID), paramProfile)
}

// getParamNameFromID returns the parameter's name, whether a parameter with ID exists, or any error.
func getParamNameFromID(id int64, tx *sql.Tx) (string, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT name from parameter where id = $1`, id).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, errors.New("querying param name from id: " + err.Error())
	}
	return name, true, nil
}

func insertParameterProfile(post tc.PostParamProfile, tx *sql.Tx) error {
	if *post.Replace {
		if _, err := tx.Exec(`DELETE FROM profile_parameter WHERE parameter = $1`, post.ParamID); err != nil {
			return errors.New("deleting old parameter profile: " + err.Error())
		}
	}
	q := `
INSERT INTO profile_parameter (profile, parameter)
VALUES (unnest($1::int[]), $2)
ON CONFLICT DO NOTHING;
`
	if _, err := tx.Exec(q, pq.Array(post.ProfileIDs), post.ParamID); err != nil {
		return errors.New("inserting parameter profile: " + err.Error())
	}
	return nil
}
