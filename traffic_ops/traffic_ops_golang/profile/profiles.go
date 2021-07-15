// Package profile includes logic and handlers for Profile-related API
// endpoints, including /profiles, /profiles/name/{{name}}/parameters,
// /profiles/{{ID}}/parameters,
// /profiles/name/{{New Profile Name}}/copy/{{existing Profile Name}},
// /profiles/import, and /profiles/{{ID}}/export.
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
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/parameter"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
)

// Supported (non-pagination) query string parameters for /profiles.
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
	api.APIInfoImpl `json:"-"`
	tc.ProfileNullable
}

func (v *TOProfile) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.APIInfo().Tx, *v.ID, "profile")
}

func (v *TOProfile) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOProfile) InsertQuery() string           { return insertQuery() }
func (v *TOProfile) UpdateQuery() string           { return updateQuery() }
func (v *TOProfile) DeleteQuery() string           { return deleteQuery() }

func (prof TOProfile) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: IDQueryParam, Func: api.GetIntKey}}
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
		NameQueryParam: validation.Validate(prof.Name, validation.By(
			func(value interface{}) error {
				name, ok := value.(*string)
				if !ok {
					return fmt.Errorf("wrong type, need: string, got: %T", value)
				}
				if name == nil || *name == "" {
					return errors.New("required and cannot be blank")
				}
				if strings.Contains(*name, " ") {
					return errors.New("cannot contain spaces")
				}
				return nil
			},
		)),
		DescriptionQueryParam: validation.Validate(prof.Description, validation.Required),
		CDNQueryParam:         validation.Validate(prof.CDNID, validation.Required),
		TypeQueryParam:        validation.Validate(prof.Type, validation.Required),
	}
	if errs != nil {
		return util.JoinErrs(tovalidate.ToErrors(errs))
	}
	return nil
}

func (prof *TOProfile) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		CDNQueryParam:  dbhelpers.WhereColumnInfo{Column: "c.id"},
		NameQueryParam: dbhelpers.WhereColumnInfo{Column: "prof.name"},
		IDQueryParam:   dbhelpers.WhereColumnInfo{Column: "prof.id", Checker: api.IsInt},
	}
	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(prof.APIInfo().Params, queryParamsToQueryCols)

	// Narrow down if the query parameter is 'param'

	// TODO add generic where clause to api.GenericRead
	if paramValue, ok := prof.APIInfo().Params[ParamQueryParam]; ok {
		queryValues["parameter_id"] = paramValue
		if len(paramValue) > 0 {
			where += " LEFT JOIN profile_parameter pp ON prof.id  = pp.profile where pp.parameter=:parameter_id"
		}
	}

	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest, nil
	}

	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(prof.APIInfo().Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return []interface{}{}, nil, nil, http.StatusNotModified, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	query := selectProfilesQuery() + where + orderBy + pagination
	log.Debugln("Query is ", query)

	rows, err := prof.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, errors.New("profile read querying: " + err.Error()), http.StatusInternalServerError, nil
	}
	defer rows.Close()

	profiles := []tc.ProfileNullable{}

	for rows.Next() {
		var p tc.ProfileNullable
		if err = rows.StructScan(&p); err != nil {
			return nil, nil, errors.New("profile read scanning: " + err.Error()), http.StatusInternalServerError, nil
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
				return nil, nil, errors.New("profile read reading parameters: " + err.Error()), http.StatusInternalServerError, nil
			}
		}
		profileInterfaces = append(profileInterfaces, profile)
	}

	return profileInterfaces, nil, nil, http.StatusOK, &maxTime

}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(prof.last_updated) as t FROM profile prof
LEFT JOIN cdn c ON prof.cdn = c.id ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='profile') as res`
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
	privLevel := user.PrivLevel
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
		if isSecure && (privLevel < auth.PrivLevelAdmin) {
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

func (pr *TOProfile) checkIfProfileCanBeAlteredByCurrentUser() (error, error, int) {
	var cdnName string
	if pr.CDNName != nil {
		cdnName = *pr.CDNName
	} else {
		if pr.CDNID != nil {
			cdn, ok, err := dbhelpers.GetCDNNameFromID(pr.ReqInfo.Tx.Tx, int64(*pr.CDNID))
			if err != nil {
				return nil, err, http.StatusInternalServerError
			} else if !ok {
				return nil, nil, http.StatusNotFound
			}
			cdnName = string(cdn)
		} else {
			return errors.New("no cdn found for this profile"), nil, http.StatusBadRequest
		}
	}
	userErr, sysErr, statusCode := dbhelpers.CheckIfCurrentUserCanModifyCDN(pr.ReqInfo.Tx.Tx, cdnName, pr.ReqInfo.User.UserName)
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, statusCode
	}
	return nil, nil, http.StatusOK
}

func (pr *TOProfile) Update(h http.Header) (error, error, int) {
	if pr.CDNName != nil || pr.CDNID != nil {
		userErr, sysErr, statusCode := pr.checkIfProfileCanBeAlteredByCurrentUser()
		if userErr != nil || sysErr != nil {
			return userErr, sysErr, statusCode
		}
	}
	return api.GenericUpdate(h, pr)
}

func (pr *TOProfile) Create() (error, error, int) {
	if pr.CDNName != nil || pr.CDNID != nil {
		userErr, sysErr, statusCode := pr.checkIfProfileCanBeAlteredByCurrentUser()
		if userErr != nil || sysErr != nil {
			return userErr, sysErr, statusCode
		}
	}
	return api.GenericCreate(pr)
}

func (pr *TOProfile) Delete() (error, error, int) {
	if pr.CDNName == nil && pr.CDNID == nil {
		cdnName, err := dbhelpers.GetCDNNameFromProfileID(pr.APIInfo().Tx.Tx, *pr.ID)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}
		pr.CDNName = util.StrPtr(string(cdnName))
	}
	if pr.CDNName != nil || pr.CDNID != nil {
		userErr, sysErr, statusCode := pr.checkIfProfileCanBeAlteredByCurrentUser()
		if userErr != nil || sysErr != nil {
			return userErr, sysErr, statusCode
		}
	}
	return api.GenericDelete(pr)
}

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
