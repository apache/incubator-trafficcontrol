package division

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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	validation "github.com/go-ozzo/ozzo-validation"
)

//we need a type alias to define functions on
type TODivision struct {
	api.InfoerImpl `json:"-"`
	tc.DivisionNullable
}

func (v *TODivision) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.Info().Tx, *v.ID, "division")
}

func (v *TODivision) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TODivision) InsertQuery() string           { return insertQuery() }
func (v *TODivision) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` d ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}

func (v *TODivision) NewReadObj() interface{} { return &tc.Division{} }
func (v *TODivision) SelectQuery() string     { return selectQuery() }
func (v *TODivision) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"id":   dbhelpers.WhereColumnInfo{Column: "id", Checker: api.IsInt},
		"name": dbhelpers.WhereColumnInfo{Column: "name"},
	}
}
func (v *TODivision) UpdateQuery() string { return updateQuery() }
func (v *TODivision) DeleteQuery() string { return deleteQuery() }

func (division TODivision) GetAuditName() string {
	if division.Name != nil {
		return *division.Name
	}
	if division.ID != nil {
		return strconv.Itoa(*division.ID)
	}
	return "unknown"
}

func (division TODivision) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (division TODivision) GetKeys() (map[string]interface{}, bool) {
	if division.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *division.ID}, true
}

func (division *TODivision) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	division.ID = &i
}

func (division TODivision) GetType() string {
	return "division"
}

func (division TODivision) Validate() error {
	errs := validation.Errors{
		"name": validation.Validate(division.Name, validation.NotNil, validation.Required),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (dv *TODivision) Create() (error, error, int) { return api.GenericCreate(dv) }
func (dv *TODivision) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	params := dv.Info().Params
	// TODO move to router, and do for all endpoints
	if strings.HasSuffix(params["name"], ".json") {
		params["name"] = params["name"][:len(params["name"])-len(".json")]
	}
	api.DefaultSort(dv.Info(), "name")
	return api.GenericRead(h, dv, useIMS)
}
func (dv *TODivision) Update(h http.Header) (error, error, int) { return api.GenericUpdate(h, dv) }
func (dv *TODivision) Delete() (error, error, int)              { return api.GenericDelete(dv) }

func insertQuery() string {
	return `INSERT INTO division (name) VALUES (:name) RETURNING id,last_updated`
}

func selectQuery() string {
	return `SELECT
id,
last_updated,
name
FROM division d`
}

func updateQuery() string {
	return `UPDATE
division SET
name=:name
WHERE id=:id RETURNING last_updated`
}

func deleteQuery() string {
	return `DELETE FROM division WHERE id=:id`
}
