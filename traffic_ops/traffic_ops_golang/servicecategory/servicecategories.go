package servicecategory

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
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	validation "github.com/go-ozzo/ozzo-validation"
)

type TOServiceCategory struct {
	api.InfoerImpl `json:"-"`
	tc.ServiceCategory
}

func (v *TOServiceCategory) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdatedByName(v.Info().Tx, v.Name, "service_category")
}

func (v *TOServiceCategory) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = t }
func (v *TOServiceCategory) InsertQuery() string           { return insertQuery() }
func (v *TOServiceCategory) NewReadObj() interface{}       { return &tc.ServiceCategory{} }
func (v *TOServiceCategory) SelectQuery() string           { return selectQuery() }
func (v *TOServiceCategory) UpdateQuery() string           { return updateQuery() }
func (v *TOServiceCategory) DeleteQuery() string           { return deleteQuery() }

func (serviceCategory TOServiceCategory) GetAuditName() string {
	if serviceCategory.Name != "" {
		return serviceCategory.Name
	}
	return "unknown"
}

func (serviceCategory TOServiceCategory) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "name", Func: api.GetStringKey}}
}

//Implementation of the Identifier, Validator interface functions
func (serviceCategory TOServiceCategory) GetKeys() (map[string]interface{}, bool) {
	if serviceCategory.Name == "" {
		return map[string]interface{}{"name": ""}, false
	}
	return map[string]interface{}{"name": serviceCategory.Name}, true
}

func (serviceCategory *TOServiceCategory) SetKeys(keys map[string]interface{}) {
	n, _ := keys["name"].(string)
	serviceCategory.Name = n
}

func (serviceCategory TOServiceCategory) GetType() string {
	return "serviceCategory"
}

func (serviceCategory *TOServiceCategory) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"name": dbhelpers.WhereColumnInfo{Column: "sc.name"},
	}
}

func (serviceCategory *TOServiceCategory) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from service_category sc ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='service_category') as res`
}

func (serviceCategory TOServiceCategory) Validate() error {
	nameRule := validation.NewStringRule(tovalidate.IsAlphanumericDash, "must consist of only alphanumeric or dash characters.")
	errs := validation.Errors{
		"name": validation.Validate(serviceCategory.Name, validation.Required, nameRule),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (serviceCategory *TOServiceCategory) Create() (error, error, int) {
	return api.GenericCreate(serviceCategory)
}

func (serviceCategory *TOServiceCategory) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	api.DefaultSort(serviceCategory.Info(), "name")
	serviceCategories, userErr, sysErr, errCode, maxTime := api.GenericRead(h, serviceCategory, useIMS)
	if userErr != nil || sysErr != nil {
		return nil, userErr, sysErr, errCode, nil
	}

	return serviceCategories, nil, nil, errCode, maxTime
}

func Update(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"name"}, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	name := inf.Params["name"]

	var newSC TOServiceCategory
	if err := json.NewDecoder(r.Body).Decode(&newSC); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	if err := newSC.Validate(); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	var origSC TOServiceCategory
	if err := inf.Tx.QueryRow(`SELECT name, last_updated FROM service_category WHERE name = $1`, name).Scan(&origSC.Name, &origSC.LastUpdated); err != nil {
		if err == sql.ErrNoRows {
			api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("no service category found with name "+name), nil)
			return
		}
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, err)
		return
	}
	if !api.IsUnmodified(r.Header, origSC.LastUpdated.Time) {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusPreconditionFailed, errors.New("service category could not be modified because the precondition failed"), nil)
		return
	}

	resp, err := inf.Tx.Tx.Exec(updateQuery(), newSC.Name, name)
	if err != nil {
		userErr, sysErr, errCode = api.ParseDBError(err)
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.CreateChangeLogRawTx(api.ApiChange, api.Updated+" Service Category from "+name+" to "+newSC.Name, inf.User, inf.Tx.Tx)
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Service Category update from "+name+" to "+newSC.Name+" was successful.", resp)
}

func (serviceCategory *TOServiceCategory) Delete() (error, error, int) {
	return api.GenericDelete(serviceCategory)
}

func insertQuery() string {
	return `INSERT INTO service_category (name) VALUES (:name) RETURNING name, last_updated`
}

func selectQuery() string {
	return `SELECT
sc.last_updated,
sc.name
FROM service_category as sc`
}

func updateQuery() string {
	return `UPDATE
service_category SET
name=$1
WHERE name=$2 RETURNING last_updated`
}

func deleteQuery() string {
	return `DELETE FROM service_category WHERE name=:name`
}
