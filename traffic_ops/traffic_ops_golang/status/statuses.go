package status

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

//we need a type alias to define functions on
type TOStatus struct {
	api.InfoerImpl `json:"-"`
	tc.StatusNullable
	SQLDescription sql.NullString `json:"-" db:"description"`
}

func (v *TOStatus) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.Info().Tx, *v.ID, "status")
}

func (v *TOStatus) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` s ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}

func (v *TOStatus) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOStatus) InsertQuery() string           { return insertQuery() }
func (v *TOStatus) NewReadObj() interface{}       { return &TOStatus{} }
func (v *TOStatus) SelectQuery() string           { return selectQuery() }
func (v *TOStatus) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"id":          dbhelpers.WhereColumnInfo{Column: "id", Checker: api.IsInt},
		"description": dbhelpers.WhereColumnInfo{Column: "description"},
		"name":        dbhelpers.WhereColumnInfo{Column: "name"},
	}
}
func (v *TOStatus) UpdateQuery() string { return updateQuery() }
func (v *TOStatus) DeleteQuery() string { return deleteQuery() }

func (status TOStatus) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (status TOStatus) GetKeys() (map[string]interface{}, bool) {
	if status.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *status.ID}, true
}

func (status *TOStatus) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	status.ID = &i
}

func (status TOStatus) GetAuditName() string {
	if status.Name != nil {
		return *status.Name
	}
	if status.ID != nil {
		return strconv.Itoa(*status.ID)
	}
	return "unknown"
}

func (status TOStatus) GetType() string { return "status" }

func (status TOStatus) Validate() error {
	errs := validation.Errors{
		"name": validation.Validate(status.Name, validation.NotNil, validation.Required),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (st *TOStatus) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	errCode := http.StatusOK
	api.DefaultSort(st.Info(), "name")
	readVals, userErr, sysErr, errCode, maxTime := api.GenericRead(h, st, useIMS)
	if userErr != nil || sysErr != nil {
		return nil, userErr, sysErr, errCode, nil
	}

	for _, iStatus := range readVals {
		status, ok := iStatus.(*TOStatus)
		if !ok {
			return nil, nil, fmt.Errorf("TOStatus.Read: api.GenericRead returned unexpected type %T\n", iStatus), http.StatusInternalServerError, nil
		}
		if status.SQLDescription.Valid {
			status.Description = &status.SQLDescription.String
		}
	}

	return readVals, nil, nil, errCode, maxTime
}

func (st *TOStatus) Update(h http.Header) (error, error, int) { return api.GenericUpdate(h, st) }
func (st *TOStatus) Create() (error, error, int)              { return api.GenericCreate(st) }
func (st *TOStatus) Delete() (error, error, int)              { return api.GenericDelete(st) }

func selectQuery() string {
	return `
SELECT
  description,
  id,
  last_updated,
  name
FROM
  status s
`
}

func updateQuery() string {
	return `UPDATE
status SET
name=:name,
description=:description
WHERE id=:id RETURNING last_updated`
}

func insertQuery() string {
	return `INSERT INTO status (
name,
description) VALUES (
:name,
:description) RETURNING id,last_updated`
}

func deleteQuery() string {
	return `DELETE FROM status WHERE id=:id`
}
