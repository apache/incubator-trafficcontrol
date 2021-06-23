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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/lib/pq"
)

func PostProfileParam(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	profileParam := tc.PostProfileParam{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &profileParam); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parse error: "+err.Error()), nil)
		return
	}
	if err := insertProfileParameter(profileParam, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("posting profile parameter: "+err.Error()))
		return
	}
	profileName, ok, err := dbhelpers.GetProfileNameFromID(int(*profileParam.ProfileID), inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting profile name from id: "+err.Error()))
		return
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("profile not found"), nil)
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, "PROFILE: "+profileName+", ID: "+strconv.FormatInt(*profileParam.ProfileID, 10)+", ACTION: Assigned "+strconv.Itoa(len(*profileParam.ParamIDs))+" parameters to profile", inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, fmt.Sprintf("%d parameters were assigned to the %s profile", len(*profileParam.ParamIDs), profileName), profileParam)
}

func insertProfileParameter(post tc.PostProfileParam, tx *sql.Tx) error {
	if *post.Replace {
		if _, err := tx.Exec(`DELETE FROM profile_parameter WHERE profile = $1`, *post.ProfileID); err != nil {
			return errors.New("deleting old profile parameter: " + err.Error())
		}
	}
	q := `
INSERT INTO profile_parameter (profile, parameter)
VALUES ($1, unnest($2::int[]))
ON CONFLICT DO NOTHING;
`
	if _, err := tx.Exec(q, *post.ProfileID, pq.Array(*post.ParamIDs)); err != nil {
		return errors.New("inserting profile parameter: " + err.Error())
	}
	return nil
}
