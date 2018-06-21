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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
)

//     $self->db->resultset('ProfileParameter')->search( \%criteria, { prefetch => [ 'parameter', 'profile' ] } )->get_column('parameter')->all();

// my $rs_data = $self->db->resultset("Parameter")->search( 'me.id' => { 'not in' => \@assigned_params } );

func GetUnassigned(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, intParams, userErr, sysErr, errCode := api.AllParams(r, []string{"id"}, []string{"id"})
		if userErr != nil || sysErr != nil {
			api.HandleErr(w, r, errCode, userErr, sysErr)
			return
		}
		api.RespWriter(w, r)(getUnassignedParametersByProfileID(intParams["id"], db))
	}
}

func getUnassignedParametersByProfileID(profileID int, db *sql.DB) ([]tc.ProfileParameterByName, error) {
	q := `
SELECT
parameter.id, parameter.name, parameter.value, parameter.config_file, parameter.secure, parameter.last_updated
FROM parameter WHERE id NOT IN (SELECT parameter FROM profile_parameter as pp WHERE pp.profile = $1)
`
	rows, err := db.Query(q, profileID)
	if err != nil {
		return nil, errors.New("querying profile name parameters: " + err.Error())
	}
	defer rows.Close()
	params := []tc.ProfileParameterByName{}
	for rows.Next() {
		p := tc.ProfileParameterByName{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Value, &p.ConfigFile, &p.Secure, &p.LastUpdated); err != nil {
			return nil, errors.New("scanning profile id parameters: " + err.Error())
		}
		params = append(params, p)
	}
	return params, nil
}
