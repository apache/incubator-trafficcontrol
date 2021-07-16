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
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	validation "github.com/go-ozzo/ozzo-validation"
)

// TOType is a needed type alias to define functions on.
type TOType struct {
	api.APIInfoImpl `json:"-"`
	tc.TypeNullable
}

func (v *TOType) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.APIInfo().Tx, *v.ID, "type")
}

func (v *TOType) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOType) InsertQuery() string           { return insertQuery() }
func (v *TOType) NewReadObj() interface{}       { return &tc.TypeNullable{} }
func (v *TOType) SelectQuery() string           { return selectQuery() }
func (v *TOType) SelectMaxLastUpdatedQuery(where string, orderBy string, pagination string, tableName string) string {
	return selectMaxLastUpdatedQuery(where, orderBy, pagination, tableName)
}
func (v *TOType) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"name":       dbhelpers.WhereColumnInfo{Column: "typ.name"},
		"id":         dbhelpers.WhereColumnInfo{Column: "typ.id", Checker: api.IsInt},
		"useInTable": dbhelpers.WhereColumnInfo{Column: "typ.use_in_table"},
	}
}
func (v *TOType) UpdateQuery() string { return updateQuery() }
func (v *TOType) DeleteQuery() string { return deleteQuery() }

func (typ TOType) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (typ TOType) GetKeys() (map[string]interface{}, bool) {
	if typ.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *typ.ID}, true
}

func (typ *TOType) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	typ.ID = &i
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

func (typ *TOType) Validate() error {
	errs := validation.Errors{
		"name":         validation.Validate(typ.Name, validation.Required),
		"description":  validation.Validate(typ.Description, validation.Required),
		"use_in_table": validation.Validate(typ.UseInTable, validation.Required),
	}
	if errs != nil {
		return util.JoinErrs(tovalidate.ToErrors(errs))
	}
	return nil
}

func (tp *TOType) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(tp.APIInfo(), "name")
	return api.GenericRead(h, tp, useIMS)
}

func (tp *TOType) Update(h http.Header) (error, error, int) {
	if !tp.AllowMutation(false) {
		return errors.New("can not update type"), nil, http.StatusBadRequest
	}
	return api.GenericUpdate(h, tp)
}

func (tp *TOType) Delete() (error, error, int) {
	if !tp.AllowMutation(false) {
		return errors.New(fmt.Sprintf("can not delete type")), nil, http.StatusBadRequest
	}
	return api.GenericDelete(tp)
}

func (tp *TOType) Create() (error, error, int) {
	if !tp.AllowMutation(true) {
		return errors.New("can not create type"), nil, http.StatusBadRequest
	}
	return api.GenericCreate(tp)
}

func (tp *TOType) AllowMutation(forCreation bool) bool {
	if !forCreation {
		userErr, sysErr, actualUseInTable := tp.loadUseInTable()
		if actualUseInTable == "" {
			return true
		}
		if userErr != nil || sysErr != nil {
			return false
		} else if actualUseInTable != "server" {
			return false
		}
	} else if *tp.UseInTable != "server" { // Only allow creating of types with UseInTable being "server"
		return false
	}
	return true
}

func (tp *TOType) loadUseInTable() (error, error, string) {
	var useInTable string
	// ID is only nil on creation, should not call this method in that case
	if tp.ID != nil {
		query := `SELECT use_in_table from type where id=$1`
		err := tp.ReqInfo.Tx.Tx.QueryRow(query, tp.ID).Scan(&useInTable)
		if err == sql.ErrNoRows {
			if tp.UseInTable == nil {
				return nil, nil, ""
			}
			return nil, nil, *tp.UseInTable
		}
		if err != nil {
			return nil, err, ""
		}
	} else {
		return errors.New("no type with that key found"), nil, ""
	}

	return nil, nil, useInTable
}

func selectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` typ ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='type') as res`
}

func selectQuery() string {
	return `SELECT
id,
name,
description,
use_in_table,
last_updated
FROM type typ`
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
