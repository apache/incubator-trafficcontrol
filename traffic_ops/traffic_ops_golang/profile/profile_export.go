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
	"errors"
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/jmoiron/sqlx"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

// ExportProfileHandler exports a profile per ID
func ExportProfileHandler(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{IDQueryParam}, []string{IDQueryParam})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	profileID := inf.IntParams[IDQueryParam]

	// Check that the profile attempting to be exported exists
	_, exists, err := dbhelpers.GetProfileNameFromID(profileID, inf.Tx.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !exists {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("profile does not exist"), nil)
		return
	}

	// Get Profile Response
	exportedProfileResp, err := getExportProfileResponse(profileID, inf.Tx)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v.json\"", *exportedProfileResp.Profile.Name))
	api.WriteRespRaw(w, r, exportedProfileResp)
}

func getExportProfileResponse(profileID int, tx *sqlx.Tx) (*tc.ProfileExportResponse, error) {
	queryValues := map[string]interface{}{
		IDQueryParam: profileID,
	}
	query := selectExportProfileQuery()
	log.Debugln("Query is ", query)

	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, errors.New("querying profile: " + err.Error())
	}
	defer rows.Close()

	type epR struct {
		ProfileDescription  *string `db:"profile_description"`
		ProfileName         *string `db:"profile_name"`
		ProfileType         *string `db:"profile_type"`
		CDN                 *string `db:"cdn"`
		ParameterName       *string `db:"parm_name"`
		ParameterConfigFile *string `db:"parm_config_file"`
		ParameterValue      *string `db:"parm_value"`
	}

	exportedProfileResp := &tc.ProfileExportResponse{}

	hasNext := rows.Next()
	if !hasNext {
		return exportedProfileResp, nil
	}
	var r epR
	if err = rows.StructScan(&r); err != nil {
		return nil, errors.New("profile read scanning: " + err.Error())
	}

	for hasNext { // Set Profile
		exportedProfileResp.Profile = tc.ProfileExportImportNullable{
			Name:        r.ProfileName,
			Description: r.ProfileDescription,
			Type:        r.ProfileType,
			CDNName:     r.CDN,
		}
		exportedProfileResp.Parameters = []tc.ProfileExportImportParameterNullable{}
		for hasNext { // Loop through parameters
			if r.ParameterName != nil {
				exportedProfileResp.Parameters = append(exportedProfileResp.Parameters,
					tc.ProfileExportImportParameterNullable{
						ConfigFile: r.ParameterConfigFile,
						Name:       r.ParameterName,
						Value:      r.ParameterValue,
					})
			}
			hasNext = rows.Next()
			if hasNext {
				if err = rows.StructScan(&r); err != nil {
					return nil, errors.New("profile read scanning: " + err.Error())
				}
			}
		}
	}

	return exportedProfileResp, nil
}

func selectExportProfileQuery() string {
	query := `SELECT
prof.description as profile_description,
prof.name as profile_name,
prof.type as profile_type,
c.name as cdn,
parm.name as parm_name,
parm.config_file as parm_config_file,
parm.value as parm_value
FROM profile prof
JOIN cdn c ON prof.cdn = c.id
LEFT JOIN profile_parameter as pp ON pp.profile = prof.id
LEFT JOIN parameter as parm ON parm.id = pp.parameter
WHERE prof.id=:id`
	return query
}
