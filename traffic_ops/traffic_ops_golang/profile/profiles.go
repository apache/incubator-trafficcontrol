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
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/parameter"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
)

const (
	CDNQueryParam         = "cdn"
	DescriptionQueryParam = "description"
	IDQueryParam          = "id"
	NameQueryParam        = "name"
	ParamQueryParam       = "param"
	TypeQueryParam        = "type"
)

//we need a type alias to define functions on
type TOProfile struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.ProfileNullable
}

func (v *TOProfile) APIInfo() *api.APIInfo         { return v.ReqInfo }
func (v *TOProfile) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOProfile) InsertQuery() string           { return insertQuery() }
func (v *TOProfile) UpdateQuery() string           { return updateQuery() }
func (v *TOProfile) DeleteQuery() string           { return deleteQuery() }

func GetTypeSingleton() api.CRUDFactory {
	return func(reqInfo *api.APIInfo) api.CRUDer {
		toReturn := TOProfile{reqInfo, tc.ProfileNullable{}}
		return &toReturn
	}
}

func (prof TOProfile) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{IDQueryParam, api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (prof TOProfile) GetKeys() (map[string]interface{}, bool) {
	if prof.ID == nil {
		return map[string]interface{}{IDQueryParam: 0}, false
	}
	return map[string]interface{}{IDQueryParam: *prof.ID}, true
}

func (prof *TOProfile) SetKeys(keys map[string]interface{}) {
	i, _ := keys[IDQueryParam].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	prof.ID = &i
}

func (prof *TOProfile) GetAuditName() string {
	if prof.Name != nil {
		return *prof.Name
	}
	if prof.ID != nil {
		return strconv.Itoa(*prof.ID)
	}
	return "unknown"
}

func (prof *TOProfile) GetType() string {
	return "profile"
}

func (prof *TOProfile) Validate() error {
	errs := validation.Errors{
		NameQueryParam:        validation.Validate(prof.Name, validation.Required),
		DescriptionQueryParam: validation.Validate(prof.Description, validation.Required),
		CDNQueryParam:         validation.Validate(prof.CDNID, validation.Required),
		TypeQueryParam:        validation.Validate(prof.Type, validation.Required),
	}
	if errs != nil {
		return util.JoinErrs(tovalidate.ToErrors(errs))
	}
	return nil
}

func (prof *TOProfile) Read() ([]interface{}, error, error, int) {
	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		CDNQueryParam:  dbhelpers.WhereColumnInfo{"c.id", nil},
		NameQueryParam: dbhelpers.WhereColumnInfo{"prof.name", nil},
		IDQueryParam:   dbhelpers.WhereColumnInfo{"prof.id", api.IsInt},
	}
	where, orderBy, queryValues, errs := dbhelpers.BuildWhereAndOrderBy(prof.APIInfo().Params, queryParamsToQueryCols)

	// Narrow down if the query parameter is 'param'

	// TODO add generic where clause to api.GenericRead
	if paramValue, ok := prof.APIInfo().Params[ParamQueryParam]; ok {
		queryValues["parameter_id"] = paramValue
		if len(paramValue) > 0 {
			where += " LEFT JOIN profile_parameter pp ON prof.id  = pp.profile where pp.parameter=:parameter_id"
		}
	}

	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest
	}

	query := selectProfilesQuery() + where + orderBy
	log.Debugln("Query is ", query)

	rows, err := prof.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, errors.New("profile read querying: " + err.Error()), http.StatusInternalServerError
	}
	defer rows.Close()

	profiles := []tc.ProfileNullable{}

	for rows.Next() {
		var p tc.ProfileNullable
		if err = rows.StructScan(&p); err != nil {
			return nil, nil, errors.New("profile read scanning: " + err.Error()), http.StatusInternalServerError
		}
		profiles = append(profiles, p)
	}
	rows.Close()
	profileInterfaces := []interface{}{}
	for _, profile := range profiles {
		// Attach Parameters if the 'id' parameter is sent
		if _, ok := prof.APIInfo().Params[IDQueryParam]; ok {
			profile.Parameters, err = ReadParameters(prof.ReqInfo.Tx, prof.APIInfo().Params, prof.ReqInfo.User, profile)
			if err != nil {
				return nil, nil, errors.New("profile read reading parameters: " + err.Error()), http.StatusInternalServerError
			}
		}
		profileInterfaces = append(profileInterfaces, profile)
	}

	return profileInterfaces, nil, nil, http.StatusOK

}

func selectProfilesQuery() string {

	query := `SELECT
prof.description,
prof.id,
prof.last_updated,
prof.name,
prof.routing_disabled,
prof.type,
c.id as cdn,
c.name as cdn_name
FROM profile prof
LEFT JOIN cdn c ON prof.cdn = c.id`

	return query
}

func ReadParameters(tx *sqlx.Tx, parameters map[string]string, user *auth.CurrentUser, profile tc.ProfileNullable) ([]tc.ParameterNullable, error) {
	queryValues := make(map[string]interface{})
	queryValues["profile_id"] = *profile.ID

	query := selectParametersQuery()
	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, errors.New("querying: " + err.Error())
	}
	defer rows.Close()

	var params []tc.ParameterNullable
	for rows.Next() {
		var param tc.ParameterNullable

		if err = rows.StructScan(&param); err != nil {
			return nil, errors.New("scanning: " + err.Error())
		}
		var isSecure bool
		if param.Secure != nil {
			isSecure = *param.Secure
		}
		if isSecure && !user.HasCapability(tc.ParameterSecureCapability) {
			param.Value = &parameter.HiddenField
		}
		params = append(params, param)
	}
	return params, nil
}

func selectParametersQuery() string {
	return `SELECT
p.id,
p.name,
p.config_file,
p.value,
p.secure
FROM parameter p
JOIN profile_parameter pp ON pp.parameter = p.id
WHERE pp.profile = :profile_id`
}

func (pr *TOProfile) Update() (error, error, int) { return api.GenericUpdate(pr) }
func (pr *TOProfile) Create() (error, error, int) { return api.GenericCreate(pr) }
func (pr *TOProfile) Delete() (error, error, int) { return api.GenericDelete(pr) }

func updateQuery() string {
	query := `UPDATE
profile SET
cdn=:cdn,
description=:description,
name=:name,
routing_disabled=:routing_disabled,
type=:type
WHERE id=:id RETURNING last_updated`
	return query
}

func insertQuery() string {
	query := `INSERT INTO profile (
cdn,
description,
name,
routing_disabled,
type) VALUES (
:cdn,
:description,
:name,
:routing_disabled,
:type) RETURNING id,last_updated`
	return query
}

func deleteQuery() string {
	return `DELETE FROM profile WHERE id = :id`
}
