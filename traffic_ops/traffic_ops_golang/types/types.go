package types

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
	"strconv"

	"github.com/apache/incubator-trafficcontrol/lib/go-log"
	"github.com/apache/incubator-trafficcontrol/lib/go-tc"
	"github.com/apache/incubator-trafficcontrol/lib/go-tc/common"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/incubator-trafficcontrol/traffic_ops/traffic_ops_golang/tovalidate"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

//we need a type alias to define functions on
type TOType tc.TypeNullable

//the refType is passed into the handlers where a copy of its type is used to decode the json.
var refType = TOType(TOType{})

func GetRefType() *TOType {
	return &refType
}

//Implementation of the Identifier, Validator interface functions
func (typ *TOType) GetID() (int, bool) {
	if typ.ID == nil {
		return 0, false
	}
	return *typ.ID, true
}

func (typ *TOType) GetAuditName() string {
	if typ.Name != nil {
		return *typ.Name
	}
	if typ.ID != nil {
		return strconv.Itoa(*typ.ID)
	}
	return "unknown"
}

func (typ *TOType) GetType() string {
	return "type"
}

func (typ *TOType) SetID(i int) {
	typ.ID = &i
}

func (typ *TOType) Validate(db *sqlx.DB) []error {
	errs := validation.Errors{
		"name":         validation.Validate(typ.Name, validation.Required),
		"description":  validation.Validate(typ.Description, validation.Required),
		"use_in_table": validation.Validate(typ.UseInTable, validation.Required),
	}
	if errs != nil {
		return tovalidate.ToErrors(errs)
	}
	return nil
}

func (typ *TOType) Read(db *sqlx.DB, parameters map[string]string, user auth.CurrentUser) ([]interface{}, []error, common.ApiErrorType) {
	var rows *sqlx.Rows

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToQueryCols := map[string]dbhelpers.WhereColumnInfo{
		"name":       dbhelpers.WhereColumnInfo{"typ.name", nil},
		"id":         dbhelpers.WhereColumnInfo{"typ.id", api.IsInt},
		"useInTable": dbhelpers.WhereColumnInfo{"typ.use_in_table", nil},
	}
	where, orderBy, queryValues, errs := dbhelpers.BuildWhereAndOrderBy(parameters, queryParamsToQueryCols)
	if len(errs) > 0 {
		return nil, errs, common.DataConflictError
	}

	query := selectQuery() + where + orderBy
	log.Debugln("Query is ", query)

	rows, err := db.NamedQuery(query, queryValues)
	if err != nil {
		log.Errorf("Error querying Types: %v", err)
		return nil, []error{common.DBError}, common.SystemError
	}
	defer rows.Close()

	types := []interface{}{}
	for rows.Next() {
		var typ TOType
		if err = rows.StructScan(&typ); err != nil {
			log.Errorf("error parsing Type rows: %v", err)
			return nil, []error{common.DBError}, common.SystemError
		}
		types = append(types, typ)
	}

	return types, []error{}, common.NoError

}

func selectQuery() string {
	query := `SELECT
id,
name,
description,
use_in_table
FROM type typ`

	return query
}

//The TOType implementation of the Updater interface
//all implementations of Updater should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if a type with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
func (typ *TOType) Update(db *sqlx.DB, user auth.CurrentUser) (error, common.ApiErrorType) {
	rollbackTransaction := true
	tx, err := db.Beginx()
	defer func() {
		if tx == nil || !rollbackTransaction {
			return
		}
		err := tx.Rollback()
		if err != nil {
			log.Errorln(errors.New("rolling back transaction: " + err.Error()))
		}
	}()

	if err != nil {
		log.Error.Printf("could not begin transaction: %v", err)
		return common.DBError, common.SystemError
	}
	log.Debugf("about to run exec query: %s with type: %++v", updateQuery(), typ)
	resultRows, err := tx.NamedQuery(updateQuery(), typ)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			err, eType := dbhelpers.ParsePQUniqueConstraintError(pqErr)
			if eType == common.DataConflictError {
				return errors.New("a type with " + err.Error()), eType
			}
			return err, eType
		}
		log.Errorf("received error: %++v from update execution", err)
		return common.DBError, common.SystemError
	}
	defer resultRows.Close()

	var lastUpdated common.TimeNoMod
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&lastUpdated); err != nil {
			log.Error.Printf("could not scan lastUpdated from insert: %s\n", err)
			return common.DBError, common.SystemError
		}
	}
	log.Debugf("lastUpdated: %++v", lastUpdated)
	typ.LastUpdated = &lastUpdated
	if rowsAffected != 1 {
		if rowsAffected < 1 {
			return errors.New("no type found with this id"), common.DataMissingError
		}
		return fmt.Errorf("this update affected too many rows: %d", rowsAffected), common.SystemError
	}
	err = tx.Commit()
	if err != nil {
		log.Errorln("Could not commit transaction: ", err)
		return common.DBError, common.SystemError
	}
	rollbackTransaction = false
	return nil, common.NoError
}

//The TOType implementation of the Creator interface
//all implementations of Creator should use transactions and return the proper errorType
//ParsePQUniqueConstraintError is used to determine if a type with conflicting values exists
//if so, it will return an errorType of DataConflict and the type should be appended to the
//generic error message returned
//The insert sql returns the id and lastUpdated values of the newly inserted type and have
//to be added to the struct
func (typ *TOType) Create(db *sqlx.DB, user auth.CurrentUser) (error, common.ApiErrorType) {
	rollbackTransaction := true
	tx, err := db.Beginx()
	defer func() {
		if tx == nil || !rollbackTransaction {
			return
		}
		err := tx.Rollback()
		if err != nil {
			log.Errorln(errors.New("rolling back transaction: " + err.Error()))
		}
	}()

	if err != nil {
		log.Error.Printf("could not begin transaction: %v", err)
		return common.DBError, common.SystemError
	}
	resultRows, err := tx.NamedQuery(insertQuery(), typ)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			err, eType := dbhelpers.ParsePQUniqueConstraintError(pqErr)
			if eType == common.DataConflictError {
				return errors.New("a type with " + err.Error()), eType
			}
			return err, eType
		}
		log.Errorf("received non pq error: %++v from create execution", err)
		return common.DBError, common.SystemError
	}
	defer resultRows.Close()

	var id int
	var lastUpdated common.TimeNoMod
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&id, &lastUpdated); err != nil {
			log.Error.Printf("could not scan id from insert: %s\n", err)
			return common.DBError, common.SystemError
		}
	}
	if rowsAffected == 0 {
		err = errors.New("no type was inserted, no id was returned")
		log.Errorln(err)
		return common.DBError, common.SystemError
	}
	if rowsAffected > 1 {
		err = errors.New("too many ids returned from type insert")
		log.Errorln(err)
		return common.DBError, common.SystemError
	}

	typ.SetID(id)
	typ.LastUpdated = &lastUpdated
	err = tx.Commit()
	if err != nil {
		log.Errorln("Could not commit transaction: ", err)
		return common.DBError, common.SystemError
	}
	rollbackTransaction = false
	return nil, common.NoError
}

//The Type implementation of the Deleter interface
//all implementations of Deleter should use transactions and return the proper errorType
func (typ *TOType) Delete(db *sqlx.DB, user auth.CurrentUser) (error, common.ApiErrorType) {
	rollbackTransaction := true
	tx, err := db.Beginx()
	defer func() {
		if tx == nil || !rollbackTransaction {
			return
		}
		err := tx.Rollback()
		if err != nil {
			log.Errorln(errors.New("rolling back transaction: " + err.Error()))
		}
	}()

	if err != nil {
		log.Error.Printf("could not begin transaction: %v", err)
		return common.DBError, common.SystemError
	}
	log.Debugf("about to run exec query: %s with type: %++v", deleteQuery(), typ)
	result, err := tx.NamedExec(deleteQuery(), typ)
	if err != nil {
		log.Errorf("received error: %++v from delete execution", err)
		return common.DBError, common.SystemError
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return common.DBError, common.SystemError
	}
	if rowsAffected < 1 {
		return errors.New("no type with that id found"), common.DataMissingError
	}
	if rowsAffected > 1 {
		return fmt.Errorf("this create affected too many rows: %d", rowsAffected), common.SystemError
	}

	err = tx.Commit()
	if err != nil {
		log.Errorln("Could not commit transaction: ", err)
		return common.DBError, common.SystemError
	}
	rollbackTransaction = false
	return nil, common.NoError
}

func updateQuery() string {
	query := `UPDATE
type SET
name=:name,
description=:description,
use_in_table=:use_in_table
WHERE id=:id RETURNING last_updated`
	return query
}

func insertQuery() string {
	query := `INSERT INTO type (
name,
description,
use_in_table) VALUES (
:name,
:description,
:use_in_table) RETURNING id,last_updated`
	return query
}

func deleteQuery() string {
	query := `DELETE FROM type
WHERE id=:id`
	return query
}
