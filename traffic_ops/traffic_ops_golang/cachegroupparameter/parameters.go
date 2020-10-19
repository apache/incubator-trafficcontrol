package cachegroupparameter

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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/jmoiron/sqlx"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/parameter"
)

const (
	CacheGroupIDQueryParam      = "id"
	CacheGroupIDNamedQueryParam = "cachegroupID"
	ParameterIDQueryParam       = "parameterId"
)

// TOCacheGroupParameter is a type alias that is used to define CRUD functions on.
type TOCacheGroupParameter struct {
	api.APIInfoImpl `json:"-"`
	tc.CacheGroupParameterNullable
	CacheGroupID int `json:"-" db:"cachegroup_id"`
}

func (cgparam *TOCacheGroupParameter) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		CacheGroupIDQueryParam: dbhelpers.WhereColumnInfo{"cgp.cachegroup", api.IsInt},
		ParameterIDQueryParam:  dbhelpers.WhereColumnInfo{"p.id", api.IsInt},
	}
}

func (cgparam *TOCacheGroupParameter) GetType() string {
	return "cachegroup parameter"
}

func (cgparam *TOCacheGroupParameter) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	queryParamsToQueryCols := cgparam.ParamColumns()
	parameters := cgparam.APIInfo().Params
	errs := apierrors.New()
	where, orderBy, pagination, queryValues, es := dbhelpers.BuildWhereAndOrderByAndPagination(parameters, queryParamsToQueryCols)
	if len(es) > 0 {
		errs.UserError = util.JoinErrs(es)
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}
	cgID, err := strconv.Atoi(parameters[CacheGroupIDQueryParam])
	if err != nil {
		errs.SetUserError("cache group id must be an integer")
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}

	_, ok, err := dbhelpers.GetCacheGroupNameFromID(cgparam.ReqInfo.Tx.Tx, cgID)
	if err != nil {
		errs.SystemError = err
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	} else if !ok {
		errs.SetUserError("cachegroup does not exist")
		errs.Code = http.StatusNotFound
		return nil, errs, nil
	}

	params := []interface{}{}
	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(cgparam.ReqInfo.Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			errs.Code = http.StatusNotModified
			return params, errs, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	query := selectQuery() + where + orderBy + pagination
	rows, err := cgparam.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		errs.SetSystemError("querying " + cgparam.GetType() + ": " + err.Error())
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	defer rows.Close()

	for rows.Next() {
		var p tc.CacheGroupParameterNullable
		if err = rows.StructScan(&p); err != nil {
			errs.SetSystemError("scanning " + cgparam.GetType() + ": " + err.Error())
			errs.Code = http.StatusInternalServerError
			return nil, errs, nil
		}
		if p.Secure != nil && *p.Secure && cgparam.ReqInfo.User.PrivLevel < auth.PrivLevelAdmin {
			p.Value = &parameter.HiddenField
		}
		params = append(params, p)
	}

	return params, errs, &maxTime
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(p.last_updated) as t FROM parameter p
LEFT JOIN cachegroup_parameter cgp ON cgp.parameter = p.id ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='parameter') as res`
}

func selectQuery() string {

	query := `SELECT
p.config_file,
p.id,
p.last_updated,
p.name,
p.value,
p.secure
FROM parameter p
LEFT JOIN cachegroup_parameter cgp ON cgp.parameter = p.id`
	return query
}

// GetKeyFieldsInfo implements the api.Identifier interface.
func (cgparam *TOCacheGroupParameter) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{
		{
			Field: CacheGroupIDNamedQueryParam,
			Func:  api.GetIntKey,
		},
		{
			Field: ParameterIDQueryParam,
			Func:  api.GetIntKey,
		},
	}
}

// SetKeys implements the api.Identifier interface and allows the
// delete handler to assign cachegroup and parameter ids.
func (cgparam *TOCacheGroupParameter) SetKeys(keys map[string]interface{}) {
	id, _ := keys[CacheGroupIDNamedQueryParam].(int)
	cgparam.CacheGroupID = id

	paramID, _ := keys[ParameterIDQueryParam].(int)
	cgparam.ID = &paramID
}

// DeleteQuery implements the api.GenericDeleter interface.
func (cgparam *TOCacheGroupParameter) DeleteQuery() string {
	return `DELETE FROM cachegroup_parameter
	WHERE cachegroup = :cachegroup_id AND parameter = :id`
}

// GetAuditName implements the api.Identifier interface.
func (cgparam *TOCacheGroupParameter) GetAuditName() string {
	if cgparam.ID != nil {
		return strconv.Itoa(cgparam.CacheGroupID) + "-" + strconv.Itoa(*cgparam.ID)
	}
	return "unknown"
}

// GetKeys implements the api.Identifier interface.
func (cgparam *TOCacheGroupParameter) GetKeys() (map[string]interface{}, bool) {
	if cgparam.ID == nil {
		return map[string]interface{}{ParameterIDQueryParam: 0}, false
	}
	return map[string]interface{}{
		CacheGroupIDNamedQueryParam: cgparam.CacheGroupID,
		ParameterIDQueryParam:       *cgparam.ID,
	}, true
}

// Delete implements the api.CRUDer interface.
func (cgparam *TOCacheGroupParameter) Delete() apierrors.Errors {
	_, ok, err := dbhelpers.GetCacheGroupNameFromID(cgparam.ReqInfo.Tx.Tx, cgparam.CacheGroupID)
	if err != nil {
		return apierrors.Errors{
			SystemError: err,
			Code:        http.StatusInternalServerError,
		}
	} else if !ok {
		return apierrors.Errors{
			UserError: fmt.Errorf("cachegroup %v does not exist", cgparam.CacheGroupID),
			Code:      http.StatusNotFound,
		}
	}

	_, ok, err = dbhelpers.GetParamNameByID(cgparam.ReqInfo.Tx.Tx, *cgparam.ID)
	if err != nil {
		return apierrors.Errors{
			SystemError: err,
			Code:        http.StatusInternalServerError,
		}
	} else if !ok {
		return apierrors.Errors{
			UserError: fmt.Errorf("parameter %v does not exist", *cgparam.ID),
			Code:      http.StatusNotFound,
		}
	}

	return api.GenericDelete(cgparam)
}

// ReadAllCacheGroupParameters reads all cachegroup parameter associations.
func ReadAllCacheGroupParameters(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()
	output, err := GetAllCacheGroupParameters(inf.Tx, inf.Params)
	if err != nil {
		api.WriteRespAlertObj(w, r, tc.ErrorLevel, "querying cachegroupparameters with error: "+err.Error(), output)
		return
	}
	api.WriteResp(w, r, output)
}

// GetAllCacheGroupParameters gets all cachegroup associations from the database and returns as slice.
func GetAllCacheGroupParameters(tx *sqlx.Tx, parameters map[string]string) (tc.CacheGroupParametersList, error) {
	if _, ok := parameters["orderby"]; !ok {
		parameters["orderby"] = "cachegroup"
	}

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"cachegroup": dbhelpers.WhereColumnInfo{"cgp.cachegroup", api.IsInt},
		"parameter":  dbhelpers.WhereColumnInfo{"cgp.parameter", api.IsInt},
	}

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(parameters, queryParamsToQueryCols)
	if len(errs) > 0 {
		return tc.CacheGroupParametersList{}, util.JoinErrs(errs)
	}

	query := selectAllQuery() + where + orderBy + pagination
	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		return tc.CacheGroupParametersList{}, errors.New("querying cachegroupParameters: " + err.Error())
	}
	defer rows.Close()

	paramsList := tc.CacheGroupParametersList{}
	params := []tc.CacheGroupParametersResponseNullable{}
	for rows.Next() {
		var p tc.CacheGroupParametersNullable
		if err = rows.Scan(&p.CacheGroup, &p.Parameter, &p.LastUpdated, &p.CacheGroupName); err != nil {
			return tc.CacheGroupParametersList{}, errors.New("scanning cachegroupParameters: " + err.Error())
		}
		params = append(params, tc.FormatForResponse(p))
	}
	paramsList.CacheGroupParameters = params
	return paramsList, nil
}

// AddCacheGroupParameters performs a Create for cachegroup parameter associations.
// AddCacheGroupParameters accepts data as a single association or an array of multiple.
func AddCacheGroupParameters(w http.ResponseWriter, r *http.Request) {
	inf, errs := api.NewInfo(r, nil, nil)
	if errs.Occurred() {
		inf.HandleErrs(w, r, errs)
		return
	}
	defer inf.Close()

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("reading request body: "+err.Error()), nil)
		return
	}

	buf := ioutil.NopCloser(bytes.NewReader(data))

	var paramsInt interface{}

	decoder := json.NewDecoder(buf)
	err = decoder.Decode(&paramsInt)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parsing json: "+err.Error()), nil)
		return
	}

	var params []tc.CacheGroupParametersNullable
	_, ok := paramsInt.([]interface{})
	var parseErr error = nil
	if !ok {
		var singleParam tc.CacheGroupParametersNullable
		parseErr = json.Unmarshal(data, &singleParam)
		if singleParam.CacheGroup == nil || singleParam.Parameter == nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("invalid cachegroup parameter."), nil)
			return
		}
		params = append(params, singleParam)
	} else {
		parseErr = json.Unmarshal(data, &params)
	}
	if parseErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parsing cachegroup parameter: "+parseErr.Error()), nil)
		return
	}

	for _, p := range params {
		ppExists, err := dbhelpers.CachegroupParameterAssociationExists(*p.Parameter, *p.CacheGroup, inf.Tx.Tx)
		if err != nil {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
			return
		}
		if ppExists {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("parameter: "+strconv.Itoa(*p.Parameter)+" already associated with cachegroup: "+strconv.Itoa(*p.CacheGroup)+"."), nil)
			return
		}
	}

	values := []string{}
	for _, param := range params {
		values = append(values, "("+strconv.Itoa(*param.CacheGroup)+", "+strconv.Itoa(*param.Parameter)+")")
	}

	insQuery := strings.Join(values, ", ")
	_, err = inf.Tx.Tx.Query(insertQuery() + insQuery)

	if err != nil {
		inf.HandleErrs(w, r, api.ParseDBError(err))
		return
	}

	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Cachegroup parameter associations were created.", params)
}

func selectAllQuery() string {
	return `SELECT cgp.cachegroup, cgp.parameter, cgp.last_updated, cg.name
				FROM cachegroup_parameter AS cgp
				JOIN cachegroup AS cg ON cg.id = cachegroup`
}

func insertQuery() string {
	return `INSERT INTO cachegroup_parameter
		(cachegroup,
		parameter)
		VALUES `
}
