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
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/parameter"
)

// TOCacheGroupUnassignedParameter Unassigned Parameter TO request
type TOCacheGroupUnassignedParameter struct {
	api.APIInfoImpl `json:"-"`
	tc.CacheGroupParameterNullable
}

// ParamColumns Parameter Where Column definitions
func (cgunparam *TOCacheGroupUnassignedParameter) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		ParameterIDQueryParam: dbhelpers.WhereColumnInfo{"p.id", api.IsInt},
	}
}

// GetType Get type string
func (cgunparam *TOCacheGroupUnassignedParameter) GetType() string {
	return "cachegroup_unassigned_params"
}

func (cgunparam TOCacheGroupUnassignedParameter) GetAuditName() string {
	if cgunparam.Name != nil {
		return *cgunparam.Name
	}
	if cgunparam.ID != nil {
		return strconv.Itoa(*cgunparam.ID)
	}
	return "unknown"
}

func (cgunparam TOCacheGroupUnassignedParameter) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

func (cgunparam TOCacheGroupUnassignedParameter) GetKeys() (map[string]interface{}, bool) {
	if cgunparam.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *cgunparam.ID}, true
}

func (cgunparam *TOCacheGroupUnassignedParameter) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	cgunparam.ID = &i
}

func (cgunparam *TOCacheGroupUnassignedParameter) Read() ([]interface{}, error, error, int) {
	queryParamsToQueryCols := cgunparam.ParamColumns()
	parameters := cgunparam.APIInfo().Params
	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(parameters, queryParamsToQueryCols)
	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest
	}

	cgID, err := strconv.Atoi(parameters[CacheGroupIDQueryParam])
	if err != nil {
		return nil, errors.New("cache group id must be an integer"), nil, http.StatusBadRequest
	}

	_, ok, err := dbhelpers.GetCacheGroupNameFromID(cgunparam.ReqInfo.Tx.Tx, int64(cgID))
	if err != nil {
		return nil, nil, err, http.StatusInternalServerError
	} else if !ok {
		return nil, errors.New("cachegroup does not exist"), nil, http.StatusNotFound
	}

	// TODO: enhance build query to handle cols that are not in WHERE as well as appending to existing WHERE
	queryValues[CacheGroupIDQueryParam] = cgID
	if len(where) > 0 {
		where = fmt.Sprintf("\nAND%s", where[len(dbhelpers.BaseWhere):])
	}

	query := selectUnassignedParametersQuery() + where + orderBy + pagination
	rows, err := cgunparam.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, errors.New("querying " + cgunparam.GetType() + ": " + err.Error()), http.StatusInternalServerError
	}
	defer rows.Close()

	params := []interface{}{}
	for rows.Next() {
		var p tc.CacheGroupParameterNullable
		if err = rows.StructScan(&p); err != nil {
			return nil, nil, errors.New("scanning " + cgunparam.GetType() + ": " + err.Error()), http.StatusInternalServerError
		}
		if p.Secure != nil && *p.Secure && cgunparam.ReqInfo.User.PrivLevel < auth.PrivLevelAdmin {
			p.Value = &parameter.HiddenField
		}
		params = append(params, p)
	}

	return params, nil, nil, http.StatusOK
}

func selectUnassignedParametersQuery() string {

	query := `SELECT
p.config_file,
p.id,
p.last_updated,
p.name,
p.value,
p.secure
FROM parameter p
WHERE p.id NOT IN (
	SELECT parameter
	FROM cachegroup_parameter as cgp
	WHERE cgp.cachegroup = :id
)`
	return query
}
