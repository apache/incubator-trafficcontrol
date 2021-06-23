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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

const API_PROFILES_NAME_NAME_PARAMETERS = "profiles/name/{name}/parameters"

func GetProfileNameDeprecated(w http.ResponseWriter, r *http.Request) {
	getProfileName(w, r, true)
}

func GetProfileName(w http.ResponseWriter, r *http.Request) {
	getProfileName(w, r, false)
}

func getProfileName(w http.ResponseWriter, r *http.Request, deprecated bool) {
	deprecation := util.StrPtr(API_PROFILES_NAME_NAME_PARAMETERS)
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, errCode, userErr, sysErr, deprecated, deprecation)
		return
	}
	defer inf.Close()

	name := inf.Params["name"]
	if deprecated {
		profiles, err := getParametersByProfileName(inf.Tx.Tx, name)
		if err != nil {
			api.HandleErrOptionalDeprecation(w, r, inf.Tx.Tx, http.StatusInternalServerError, err, nil, deprecated, deprecation)
			return
		}
		api.WriteAlertsObj(w, r, http.StatusOK, api.CreateDeprecationAlerts(deprecation), profiles)
	} else {
		api.RespWriter(w, r, inf.Tx.Tx)(getParametersByProfileName(inf.Tx.Tx, name))
	}
}

func getParametersByProfileName(tx *sql.Tx, profileName string) ([]tc.ProfileParameterByName, error) {
	q := `
SELECT
parameter.id, parameter.name, parameter.value, parameter.config_file, parameter.secure, parameter.last_updated
FROM parameter
JOIN profile_parameter as pp ON pp.parameter = parameter.id
JOIN profile on profile.id = pp.profile
WHERE profile.name = $1
`
	rows, err := tx.Query(q, profileName)
	if err != nil {
		return nil, errors.New("querying profile name parameters: " + err.Error())
	}
	defer rows.Close()
	params := []tc.ProfileParameterByName{}
	for rows.Next() {
		p := tc.ProfileParameterByName{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Value, &p.ConfigFile, &p.Secure, &p.LastUpdated); err != nil {
			return nil, errors.New("scanning profile name parameters: " + err.Error())
		}
		params = append(params, p)
	}
	return params, nil
}
