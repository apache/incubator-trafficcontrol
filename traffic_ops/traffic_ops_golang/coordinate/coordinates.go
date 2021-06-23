package coordinate

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
type TOCoordinate struct {
	api.InfoerImpl `json:"-"`
	tc.CoordinateNullable
}

func (v *TOCoordinate) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOCoordinate) InsertQuery() string           { return insertQuery() }
func (v *TOCoordinate) NewReadObj() interface{}       { return &tc.CoordinateNullable{} }
func (v *TOCoordinate) SelectQuery() string           { return selectQuery() }
func (v *TOCoordinate) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"id":   dbhelpers.WhereColumnInfo{Column: "id", Checker: api.IsInt},
		"name": dbhelpers.WhereColumnInfo{Column: "name"},
	}
}

func (v *TOCoordinate) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.Info().Tx, *v.ID, "coordinate")
}

func (v *TOCoordinate) UpdateQuery() string { return updateQuery() }
func (v *TOCoordinate) DeleteQuery() string { return deleteQuery() }
func (coordinate TOCoordinate) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (coordinate TOCoordinate) GetKeys() (map[string]interface{}, bool) {
	if coordinate.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *coordinate.ID}, true
}

func (coordinate TOCoordinate) GetAuditName() string {
	if coordinate.Name != nil {
		return *coordinate.Name
	}
	if coordinate.ID != nil {
		return strconv.Itoa(*coordinate.ID)
	}
	return "0"
}

func (coordinate TOCoordinate) GetType() string {
	return "coordinate"
}

func (coordinate *TOCoordinate) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	coordinate.ID = &i
}

func isValidCoordinateChar(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	if r == '.' || r == '-' || r == '_' {
		return true
	}
	return false
}

// IsValidCoordinateName returns true if the name contains only characters valid for a Coordinate name
func IsValidCoordinateName(str string) bool {
	i := strings.IndexFunc(str, func(r rune) bool { return !isValidCoordinateChar(r) })
	return i == -1
}

// Validate fulfills the api.Validator interface
func (coordinate TOCoordinate) Validate() error {
	validName := validation.NewStringRule(IsValidCoordinateName, "invalid characters found - Use alphanumeric . or - or _ .")
	latitudeErr := "Must be a floating point number within the range +-90"
	longitudeErr := "Must be a floating point number within the range +-180"
	errs := validation.Errors{
		"name":      validation.Validate(coordinate.Name, validation.Required, validName),
		"latitude":  validation.Validate(coordinate.Latitude, validation.Min(-90.0).Error(latitudeErr), validation.Max(90.0).Error(latitudeErr)),
		"longitude": validation.Validate(coordinate.Longitude, validation.Min(-180.0).Error(longitudeErr), validation.Max(180.0).Error(longitudeErr)),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (coord *TOCoordinate) Create() (error, error, int) { return api.GenericCreate(coord) }
func (coord *TOCoordinate) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(coord.Info(), "name")
	return api.GenericRead(h, coord, useIMS)
}
func (v *TOCoordinate) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` c ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}

func (coord *TOCoordinate) Update(h http.Header) (error, error, int) {
	return api.GenericUpdate(h, coord)
}
func (coord *TOCoordinate) Delete() (error, error, int) { return api.GenericDelete(coord) }

func selectQuery() string {
	query := `SELECT
id,
latitude,
longitude,
last_updated,
name

FROM coordinate c`
	return query
}

func updateQuery() string {
	query := `UPDATE
coordinate SET
latitude=:latitude,
longitude=:longitude,
name=:name
WHERE id=:id RETURNING last_updated`
	return query
}

func insertQuery() string {
	query := `INSERT INTO coordinate (
latitude,
longitude,
name) VALUES (
:latitude,
:longitude,
:name) RETURNING id,last_updated`
	return query
}

func deleteQuery() string {
	return `DELETE FROM coordinate WHERE id = :id`
}
