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
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

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

func (cgunparam *TOCacheGroupUnassignedParameter) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	errs := apierrors.New()
	queryParamsToQueryCols := cgunparam.ParamColumns()
	parameters := cgunparam.APIInfo().Params
	where, orderBy, pagination, queryValues, es := dbhelpers.BuildWhereAndOrderByAndPagination(parameters, queryParamsToQueryCols)
	if len(es) > 0 {
		errs.UserError = util.JoinErrs(es)
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}

	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(cgunparam.APIInfo().Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			errs.Code = http.StatusNotModified
			return []interface{}{}, errs, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	cgID, err := strconv.Atoi(parameters[CacheGroupIDQueryParam])
	if err != nil {
		errs.SetUserError("cache group id must be an integer")
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}

	_, ok, err := dbhelpers.GetCacheGroupNameFromID(cgunparam.ReqInfo.Tx.Tx, cgID)
	if err != nil {
		errs.Code = http.StatusInternalServerError
		errs.SystemError = err
		return nil, errs, nil
	} else if !ok {
		errs.Code = http.StatusNotFound
		errs.SetUserError("cachegroup does not exist")
		return nil, errs, nil
	}

	// TODO: enhance build query to handle cols that are not in WHERE as well as appending to existing WHERE
	queryValues[CacheGroupIDQueryParam] = cgID
	if len(where) > 0 {
		where = fmt.Sprintf("\nAND%s", where[len(dbhelpers.BaseWhere):])
	}

	query := selectUnassignedParametersQuery() + where + orderBy + pagination
	rows, err := cgunparam.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		errs.SystemError = errors.New("querying " + cgunparam.GetType() + ": " + err.Error())
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	defer rows.Close()

	params := []interface{}{}
	for rows.Next() {
		var p tc.CacheGroupParameterNullable
		if err = rows.StructScan(&p); err != nil {
			errs.SystemError = errors.New("scanning " + cgunparam.GetType() + ": " + err.Error())
			errs.Code = http.StatusInternalServerError
			return nil, errs, nil
		}
		if p.Secure != nil && *p.Secure && cgunparam.ReqInfo.User.PrivLevel < auth.PrivLevelAdmin {
			p.Value = &parameter.HiddenField
		}
		params = append(params, p)
	}

	return params, errs, &maxTime
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
