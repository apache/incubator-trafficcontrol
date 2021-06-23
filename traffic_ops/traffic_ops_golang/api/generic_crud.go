package api

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
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	ims "github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

type GenericCreator interface {
	GetType() string
	Info() *Info
	SetKeys(map[string]interface{})
	SetLastUpdated(tc.TimeNoMod)
	InsertQuery() string
}

type GenericReader interface {
	GetType() string
	Info() *Info
	ParamColumns() map[string]dbhelpers.WhereColumnInfo
	NewReadObj() interface{}
	SelectQuery() string
	SelectMaxLastUpdatedQuery(where string, orderBy string, pagination string, tableName string) string
}

type GenericUpdater interface {
	GetType() string
	Info() *Info
	SetLastUpdated(tc.TimeNoMod)
	UpdateQuery() string
	GetLastUpdated() (*time.Time, bool, error)
}

type GenericDeleter interface {
	GetType() string
	Info() *Info
	DeleteQuery() string
}

// GenericOptionsDeleter can use any key listed in DeleteKeyOptions() to delete a resource.
type GenericOptionsDeleter interface {
	GetType() string
	Info() *Info
	DeleteKeyOptions() map[string]dbhelpers.WhereColumnInfo
	DeleteQueryBase() string
}

// GenericCreate does a Create (POST) for the given GenericCreator object and type. This exists as a generic function, for the common use case of a single "id" key and a lastUpdated field.
func GenericCreate(val GenericCreator) (error, error, int) {
	resultRows, err := val.Info().Tx.NamedQuery(val.InsertQuery(), val)
	if err != nil {
		return ParseDBError(err)
	}
	defer resultRows.Close()

	var id interface{}
	lastUpdated := tc.TimeNoMod{}
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&id, &lastUpdated); err != nil {
			return nil, errors.New(val.GetType() + " create scanning: " + err.Error()), http.StatusInternalServerError
		}
	}

	switch id.(type) {
	case int64:
		// ahhh generics in a language without generics
		// sql.Driver values return int64s from integral database data
		// but our objects all use ambiguous-width ints for their IDs
		// so naturally this suffers from overflow issues, but c'est la vie
		id = int(id.(int64))
	default:
	}
	if rowsAffected == 0 {
		return nil, errors.New(val.GetType() + " create: no " + val.GetType() + " was inserted, no id was returned"), http.StatusInternalServerError
	} else if rowsAffected > 1 {
		return nil, errors.New("too many ids returned from " + val.GetType() + " insert"), http.StatusInternalServerError
	}
	val.SetKeys(map[string]interface{}{"id": id})
	val.SetLastUpdated(lastUpdated)
	return nil, nil, http.StatusOK
}

// GenericCreateNameBasedID does a Create (POST) for the given GenericCreator object and type. This exists as a generic function, for the use case of a single "name" key (not a numerical "id" key) and a lastUpdated field.
func GenericCreateNameBasedID(val GenericCreator) (error, error, int) {
	resultRows, err := val.Info().Tx.NamedQuery(val.InsertQuery(), val)
	if err != nil {
		return ParseDBError(err)
	}
	defer resultRows.Close()

	lastUpdated := tc.TimeNoMod{}
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&lastUpdated); err != nil {
			return nil, errors.New(val.GetType() + " create scanning: " + err.Error()), http.StatusInternalServerError
		}
	}
	if rowsAffected == 0 {
		return nil, errors.New(val.GetType() + " create: no " + val.GetType() + " was inserted, no row was returned"), http.StatusInternalServerError
	} else if rowsAffected > 1 {
		return nil, errors.New("too many rows returned from " + val.GetType() + " insert"), http.StatusInternalServerError
	}
	val.SetLastUpdated(lastUpdated)
	return nil, nil, http.StatusOK
}

// TryIfModifiedSinceQuery checks to see the max time that an entity was changed, and then returns a boolean (which tells us whether or not to run the main query for the endpoint)
// along with the max time
// If the returned boolean is false, there is no need to run the main query for the GET API endpoint, and we return a 304 status
func TryIfModifiedSinceQuery(val GenericReader, h http.Header, where string, orderBy string, pagination string, queryValues map[string]interface{}) (bool, time.Time) {
	var max time.Time
	var imsDate time.Time
	var ok bool
	imsDateHeader := []string{}
	runSecond := true
	dontRunSecond := false
	if h == nil {
		return runSecond, max
	}
	imsDateHeader = h[rfc.IfModifiedSince]
	if len(imsDateHeader) == 0 {
		return runSecond, max
	}
	if imsDate, ok = rfc.ParseHTTPDate(imsDateHeader[0]); !ok {
		log.Warnf("IMS request header date '%s' not parsable", imsDateHeader[0])
		return runSecond, max
	}
	// ToDo: Remove orderBy, pagination from all the implementations, and eventually remove it from the function definition
	query := val.SelectMaxLastUpdatedQuery(where, "", "", val.GetType())
	rows, err := val.Info().Tx.NamedQuery(query, queryValues)
	if err != nil {
		log.Warnf("Couldn't get the max last updated time: %v", err)
		return runSecond, max
	}
	if err == sql.ErrNoRows {
		return dontRunSecond, max
	}
	defer rows.Close()
	// This should only ever contain one row
	if rows.Next() {
		v := &ims.LatestTimestamp{}
		if err = rows.StructScan(v); err != nil || v == nil {
			log.Warnf("Failed to parse the max time stamp into a struct %v", err)
			return runSecond, max
		}
		if v.LatestTime != nil {
			max = v.LatestTime.Time
			// The request IMS time is later than the max of (lastUpdated, deleted_time)
			if imsDate.After(v.LatestTime.Time) {
				return dontRunSecond, max
			}
		}
	}
	return runSecond, max
}

func GenericRead(h http.Header, val GenericReader, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	vals := []interface{}{}
	code := http.StatusOK
	var maxTime time.Time
	var runSecond bool
	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(val.Info().Params, val.ParamColumns())
	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest, nil
	}
	if useIMS {
		runSecond, maxTime = TryIfModifiedSinceQuery(val, h, where, orderBy, pagination, queryValues)
		if !runSecond {
			log.Debugln("IMS HIT")
			code = http.StatusNotModified
			return vals, nil, nil, code, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}
	// Case where we need to run the second query
	query := val.SelectQuery() + where + orderBy + pagination
	rows, err := val.Info().Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, errors.New("querying " + val.GetType() + ": " + err.Error()), http.StatusInternalServerError, &maxTime
	}
	defer rows.Close()

	for rows.Next() {
		v := val.NewReadObj()
		if err = rows.StructScan(v); err != nil {
			return nil, nil, errors.New("scanning " + val.GetType() + ": " + err.Error()), http.StatusInternalServerError, &maxTime
		}
		vals = append(vals, v)
	}
	return vals, nil, nil, code, &maxTime
}

// GenericUpdate handles the common update case, where the update returns the new last_modified time.
func GenericUpdate(h http.Header, val GenericUpdater) (error, error, int) {
	existingLastUpdated, found, err := val.GetLastUpdated()
	if err == nil && found == false {
		return errors.New("no " + val.GetType() + " found with this id"), nil, http.StatusNotFound
	}
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	if !IsUnmodified(h, *existingLastUpdated) {
		return ResourceModifiedError, nil, http.StatusPreconditionFailed
	}

	rows, err := val.Info().Tx.NamedQuery(val.UpdateQuery(), val)
	if err != nil {
		return ParseDBError(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return errors.New("no " + val.GetType() + " found with this id"), nil, http.StatusNotFound
	}
	lastUpdated := tc.TimeNoMod{}
	if err := rows.Scan(&lastUpdated); err != nil {
		return nil, errors.New("scanning lastUpdated from " + val.GetType() + " insert: " + err.Error()), http.StatusInternalServerError
	}
	val.SetLastUpdated(lastUpdated)
	if rows.Next() {
		return nil, errors.New(val.GetType() + " update affected too many rows: >1"), http.StatusInternalServerError
	}
	return nil, nil, http.StatusOK
}

// GenericOptionsDelete does a Delete (DELETE) for the given GenericOptionsDeleter object and type. Unlike
// GenericDelete, there is no requirement that a specific key is used as the parameter.
// GenericOptionsDeleter.DeleteKeyOptions() specifies which keys can be used.
func GenericOptionsDelete(val GenericOptionsDeleter) (error, error, int) {
	where, _, _, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(val.Info().Params, val.DeleteKeyOptions())
	if len(errs) > 0 {
		return util.JoinErrs(errs), nil, http.StatusBadRequest
	}

	query := val.DeleteQueryBase() + where
	tx := val.Info().Tx
	result, err := tx.NamedExec(query, queryValues)
	if err != nil {
		return ParseDBError(err)
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return nil, errors.New("deleting " + val.GetType() + ": getting rows affected: " + err.Error()), http.StatusInternalServerError
	} else if rowsAffected < 1 {
		return errors.New("no " + val.GetType() + " with that key found"), nil, http.StatusNotFound
	} else if rowsAffected > 1 {
		return nil, fmt.Errorf(val.GetType()+" delete affected too many rows: %d", rowsAffected), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

// GenericDelete does a Delete (DELETE) for the given GenericDeleter object and type. This exists as a generic function, for the common use case of a simple delete with query parameters defined in the sqlx struct tags.
func GenericDelete(val GenericDeleter) (error, error, int) {
	result, err := val.Info().Tx.NamedExec(val.DeleteQuery(), val)
	if err != nil {
		return ParseDBError(err)
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return nil, errors.New("deleting " + val.GetType() + ": getting rows affected: " + err.Error()), http.StatusInternalServerError
	} else if rowsAffected < 1 {
		return errors.New("no " + val.GetType() + " with that key found"), nil, http.StatusNotFound
	} else if rowsAffected > 1 {
		return nil, fmt.Errorf(val.GetType()+" delete affected too many rows: %d", rowsAffected), http.StatusInternalServerError
	}
	return nil, nil, http.StatusOK
}
