package physlocation

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
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	validation "github.com/go-ozzo/ozzo-validation"
)

//we need a type alias to define functions on
type TOPhysLocation struct {
	api.InfoerImpl `json:"-"`
	tc.PhysLocationNullable
}

func (v *TOPhysLocation) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(v.Info().Tx, *v.ID, "phys_location")
}

func (v *TOPhysLocation) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOPhysLocation) InsertQuery() string           { return insertQuery() }
func (v *TOPhysLocation) NewReadObj() interface{}       { return &tc.PhysLocationNullable{} }
func (v *TOPhysLocation) SelectQuery() string           { return selectQuery() }
func (v *TOPhysLocation) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"name":   dbhelpers.WhereColumnInfo{Column: "pl.name"},
		"id":     dbhelpers.WhereColumnInfo{Column: "pl.id", Checker: api.IsInt},
		"region": dbhelpers.WhereColumnInfo{Column: "pl.region", Checker: api.IsInt},
	}
}
func (v *TOPhysLocation) UpdateQuery() string { return updateQuery() }
func (v *TOPhysLocation) DeleteQuery() string { return deleteQuery() }

func (pl TOPhysLocation) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (pl TOPhysLocation) GetKeys() (map[string]interface{}, bool) {
	if pl.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *pl.ID}, true
}

func (pl *TOPhysLocation) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	pl.ID = &i
}

func (pl *TOPhysLocation) GetAuditName() string {
	if pl.Name != nil {
		return *pl.Name
	}
	if pl.ID != nil {
		return strconv.Itoa(*pl.ID)
	}
	return "unknown"
}

func (pl *TOPhysLocation) GetType() string {
	return "physLocation"
}

func (pl *TOPhysLocation) Validate() error {
	errs := validation.Errors{
		"address":   validation.Validate(pl.Address, validation.Required),
		"city":      validation.Validate(pl.City, validation.Required),
		"name":      validation.Validate(pl.Name, validation.Required),
		"regionId":  validation.Validate(pl.RegionID, validation.Required, validation.Min(0)),
		"shortName": validation.Validate(pl.ShortName, validation.Required),
		"state":     validation.Validate(pl.State, validation.Required),
		"zip":       validation.Validate(pl.Zip, validation.Required),
	}
	if errs != nil {
		return util.JoinErrs(tovalidate.ToErrors(errs))
	}
	return nil
}

func (pl *TOPhysLocation) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(pl.Info(), "name")
	return api.GenericRead(h, pl, useIMS)
}
func (v *TOPhysLocation) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(pl.last_updated) as t FROM phys_location pl
JOIN region r ON pl.region = r.id ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='phys_location') as res`
}

func (pl *TOPhysLocation) Update(h http.Header) (error, error, int) { return api.GenericUpdate(h, pl) }
func (pl *TOPhysLocation) Create() (error, error, int)              { return api.GenericCreate(pl) }
func (pl *TOPhysLocation) Delete() (error, error, int)              { return api.GenericDelete(pl) }

func selectQuery() string {
	return `
SELECT
pl.address,
pl.city,
pl.comments,
pl.email,
pl.id,
pl.last_updated,
pl.name,
pl.phone,
pl.poc,
r.id as region,
r.name as region_name,
pl.short_name,
pl.state,
pl.zip
FROM phys_location pl
JOIN region r ON pl.region = r.id
`
}

func updateQuery() string {
	query := `UPDATE
phys_location SET
address=:address,
city=:city,
comments=:comments,
email=:email,
name=:name,
phone=:phone,
poc=:poc,
region=:region,
short_name=:short_name,
state=:state,
zip=:zip
WHERE id=:id RETURNING last_updated`
	return query
}

func insertQuery() string {
	query := `INSERT INTO phys_location (
address,
city,
comments,
email,
name,
phone,
poc,
region,
short_name,
state,
zip) VALUES (
:address,
:city,
:comments,
:email,
:name,
:phone,
:poc,
:region,
:short_name,
:state,
:zip) RETURNING id,last_updated`
	return query
}

func deleteQuery() string {
	query := `DELETE FROM phys_location
WHERE id=:id`
	return query
}
