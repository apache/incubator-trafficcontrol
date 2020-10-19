package steeringtargets

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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/jmoiron/sqlx"
)

type TOSteeringTargetV11 struct {
	api.APIInfoImpl `json:"-"`
	tc.SteeringTargetNullable
	DSTenantID  *int          `json:"-" db:"tenant"`
	LastUpdated *tc.TimeNoMod `json:"-" db:"last_updated"`
}

func (st TOSteeringTargetV11) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{
		{"deliveryservice", api.GetIntKey},
		{"target", api.GetIntKey},
	}
}

func (st TOSteeringTargetV11) GetKeys() (map[string]interface{}, bool) {
	keys := map[string]interface{}{}
	valid := true
	if st.DeliveryServiceID == nil {
		keys["deliveryservice"] = 0
		valid = false
	} else {
		keys["deliveryservice"] = int(*st.DeliveryServiceID)
	}
	if st.TargetID == nil {
		keys["target"] = 0
		valid = false
	} else {
		keys["target"] = int(*st.TargetID)
	}
	return keys, valid
}

func (st *TOSteeringTargetV11) SetKeys(keys map[string]interface{}) {
	dsI, _ := keys["deliveryservice"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	ds := uint64(dsI)
	st.DeliveryServiceID = &ds
	targetI, _ := keys["target"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	target := uint64(targetI)
	st.TargetID = &target
}

func (st TOSteeringTargetV11) GetAuditName() string {
	if st.DeliveryService != nil && st.Target != nil {
		return string(*st.DeliveryService) + `-` + string(*st.Target)
	}
	if st.DeliveryServiceID != nil && st.TargetID != nil {
		return strconv.FormatUint(*st.DeliveryServiceID, 10) + `-` + strconv.FormatUint(*st.TargetID, 10)
	}
	return "unknown"
}

func (st TOSteeringTargetV11) GetType() string {
	return "steeringtarget"
}

func (st TOSteeringTargetV11) Validate() error {
	return st.SteeringTargetNullable.Validate(st.ReqInfo.Tx.Tx)
}

func (st *TOSteeringTargetV11) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	steeringTargets, errs, maxTime := read(h, st.ReqInfo.Tx, st.ReqInfo.Params, st.ReqInfo.User, useIMS)
	if errs.Occurred() {
		return nil, errs, nil
	}
	iSteeringTargets := make([]interface{}, len(steeringTargets), len(steeringTargets))
	for i, steeringTarget := range steeringTargets {
		iSteeringTargets[i] = steeringTarget
	}
	return iSteeringTargets, errs, maxTime
}

func read(h http.Header, tx *sqlx.Tx, parameters map[string]string, user *auth.CurrentUser, useIMS bool) ([]tc.SteeringTargetNullable, apierrors.Errors, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"deliveryservice": dbhelpers.WhereColumnInfo{"st.deliveryservice", api.IsInt},
		"target":          dbhelpers.WhereColumnInfo{"st.target", api.IsInt},
	}
	errs := apierrors.New()
	where, orderBy, pagination, queryValues, dbErrs := dbhelpers.BuildWhereAndOrderByAndPagination(parameters, queryParamsToQueryCols)
	if len(dbErrs) > 0 {
		errs.UserError = util.JoinErrs(dbErrs)
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}

	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return []tc.SteeringTargetNullable{}, apierrors.Errors{Code: http.StatusNotModified}, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}

	query := selectQuery() + where + orderBy + pagination

	userTenants, err := tenant.GetUserTenantListTx(*user, tx.Tx)
	if err != nil {
		errs.SetSystemError("getting user tenant list: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}

	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		errs.SetSystemError("steering targets querying: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	defer rows.Close()

	steeringTargets := []TOSteeringTargetV11{}
	for rows.Next() {
		s := TOSteeringTargetV11{}
		if err = rows.StructScan(&s); err != nil {
			errs.SetSystemError("steering targets parsing: " + err.Error())
			errs.Code = http.StatusInternalServerError
			return nil, errs, nil
		}
		steeringTargets = append(steeringTargets, s)
	}

	tenantMap := map[int]struct{}{}
	for _, ten := range userTenants {
		if ten.ID == nil {
			errs.SetSystemError("user tenant with nil ID")
			errs.Code = http.StatusInternalServerError
			return nil, errs, nil
		}
		tenantMap[*ten.ID] = struct{}{}
	}

	filteredTargets := []tc.SteeringTargetNullable{}
	for _, tr := range steeringTargets {
		if tr.DSTenantID == nil {
			filteredTargets = append(filteredTargets, tr.SteeringTargetNullable)
			continue
		}
		if _, ok := tenantMap[int(*tr.DSTenantID)]; ok {
			filteredTargets = append(filteredTargets, tr.SteeringTargetNullable)
			continue
		}
	}
	return filteredTargets, errs, &maxTime
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(st.last_updated) as t FROM steering_target AS st
	JOIN deliveryservice AS ds ON st.deliveryservice = ds.id
	JOIN deliveryservice AS dst ON st.target = dst.id
	JOIN type AS tp ON tp.id = st.type ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='steering_target') as res`
}

func (st *TOSteeringTargetV11) Create() apierrors.Errors {
	errs := apierrors.Errors{
		Code: http.StatusBadRequest,
	}
	dsIDInt, err := strconv.Atoi(st.ReqInfo.Params["deliveryservice"])
	if err != nil {
		errs.SetUserError("delivery service ID must be an integer")
		return errs
	}
	dsID := uint64(dsIDInt)
	st.DeliveryServiceID = &dsID

	// target can't be in the Validate func, because it's in the parameters of PUT, not the body (but it is in the body in the POST here).
	if st.TargetID == nil {
		errs.SetUserError("missing target")
		return errs
	}

	if errs = tenant.CheckID(st.ReqInfo.Tx.Tx, st.ReqInfo.User, int(*st.DeliveryServiceID)); errs.Occurred() {
		return errs
	}

	rows, err := st.ReqInfo.Tx.NamedQuery(insertQuery(), st)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer rows.Close()

	errs = apierrors.Errors{
		Code: http.StatusInternalServerError,
	}
	rowsAffected := 0
	for rows.Next() {
		rowsAffected++
		if err = rows.StructScan(&st); err != nil {
			errs.SystemError = errors.New("steering target create scanning: " + err.Error())
			return errs
		}
	}
	if rowsAffected == 0 {
		errs.SystemError = errors.New("no " + st.GetType() + " was inserted, no id was returned")
		return errs
	} else if rowsAffected > 1 {
		errs.SetSystemError("too many ids returned from steering target insert")
		return errs
	}
	return apierrors.New()
}

func (st *TOSteeringTargetV11) Update() apierrors.Errors {
	errs := apierrors.New()
	dsIDInt, err := strconv.Atoi(st.ReqInfo.Params["deliveryservice"])
	if err != nil {
		errs.SetUserError("delivery service ID must be an integer")
		errs.Code = http.StatusBadRequest
		return errs
	}
	dsID := uint64(dsIDInt)
	// TODO determine if the CRUDer automatically does this
	st.DeliveryServiceID = &dsID

	targetIDInt, err := strconv.Atoi(st.ReqInfo.Params["target"])
	if err != nil {
		errs.SetUserError("target ID must be an integer")
		errs.Code = http.StatusBadRequest
	}
	targetID := uint64(targetIDInt)
	st.TargetID = &targetID

	if errs = tenant.CheckID(st.ReqInfo.Tx.Tx, st.ReqInfo.User, int(*st.DeliveryServiceID)); errs.Occurred() {
		return errs
	}

	rows, err := st.ReqInfo.Tx.NamedQuery(updateQuery(), st)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer rows.Close()

	lastUpdated := tc.TimeNoMod{}
	rowsAffected := 0
	for rows.Next() {
		rowsAffected++
		if err = rows.StructScan(&st); err != nil {
			errs.SystemError = errors.New("steering target update scanning: " + err.Error())
			errs.Code = http.StatusInternalServerError
			return errs
		}
	}
	st.LastUpdated = &lastUpdated
	if rowsAffected != 1 {
		if rowsAffected < 1 {
			errs.SetUserError("steering target not found")
			errs.Code = http.StatusNotFound
		} else {
			errs.SetSystemError("too many ids returned from steering target update")
			errs.Code = http.StatusInternalServerError
		}
	}
	return errs
}

func (st *TOSteeringTargetV11) Delete() apierrors.Errors {
	errs := tenant.CheckID(st.ReqInfo.Tx.Tx, st.ReqInfo.User, int(*st.DeliveryServiceID))
	if errs.Occurred() {
		return errs
	}

	result, err := st.ReqInfo.Tx.NamedExec(deleteQuery(), st)
	if err != nil {
		errs.SetSystemError("steering target delete exec: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return errs
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		errs.SetSystemError("steering target delete exec getting rows affected: " + err.Error())
		errs.Code = http.StatusInternalServerError
		return errs
	}

	if rowsAffected < 1 {
		errs.SetUserError("steering target not found")
		errs.Code = http.StatusNotFound
	} else if rowsAffected != 1 {
		errs.SystemError = fmt.Errorf("this create affected too many rows: %d", rowsAffected)
		errs.Code = http.StatusInternalServerError
	}
	return errs
}

func selectQuery() string {
	return `
SELECT
  st.deliveryservice,
  ds.xml_id as deliveryservice_name,
  CAST(ds.tenant_id AS INTEGER) as tenant,
  st.target,
  dst.xml_id AS target_name,
  st.type as type_id,
  tp.name as type_name,
  st.value
FROM steering_target AS st
JOIN deliveryservice AS ds ON st.deliveryservice = ds.id
JOIN deliveryservice AS dst ON st.target = dst.id
JOIN type AS tp ON tp.id = st.type
`
}

func insertQuery() string {
	return `
WITH st AS (
  INSERT INTO steering_target (deliveryservice, target, value, type)
  VALUES (:deliveryservice, :target, :value, :type_id)
  RETURNING deliveryservice, target, value, type
)
SELECT
  st.deliveryservice,
  ds.xml_id as deliveryservice_name,
  st.target,
  dst.xml_id AS target_name,
  st.type as type_id,
  tp.name as type_name,
  st.value
FROM st
JOIN deliveryservice AS ds ON st.deliveryservice = ds.id
JOIN deliveryservice AS dst ON st.target = dst.id
JOIN type AS tp ON tp.id = st.type
`
}

func updateQuery() string {
	return `
WITH st as (
  UPDATE steering_target SET
    value = :value,
    type = :type_id
  WHERE deliveryservice = :deliveryservice AND target = :target
  RETURNING deliveryservice, target, value, type, last_updated
)
SELECT
  st.deliveryservice,
  ds.xml_id as deliveryservice_name,
  st.target,
  dst.xml_id AS target_name,
  st.type as type_id,
  tp.name as type_name,
  st.value,
  st.last_updated
FROM st
JOIN deliveryservice AS ds ON st.deliveryservice = ds.id
JOIN deliveryservice AS dst ON st.target = dst.id
JOIN type AS tp ON tp.id = st.type
`
}

func deleteQuery() string {
	return `DELETE FROM steering_target WHERE deliveryservice = :deliveryservice AND target = :target`
}
