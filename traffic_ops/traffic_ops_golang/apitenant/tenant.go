package apitenant

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

// tenant.go defines the TOTenant object and methods/functions required for the api/.../tenants endpoints

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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/lib/pq"
)

// TOTenant provides a local type against which to define methods
type TOTenant struct {
	api.APIInfoImpl `json:"-"`
	tc.TenantNullable
}

func (v *TOTenant) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOTenant) InsertQuery() string           { return insertQuery() }
func (v *TOTenant) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` q ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}
func (v *TOTenant) NewReadObj() interface{} { return &tc.TenantNullable{} }
func (v *TOTenant) SelectQuery() string {
	return selectQuery(v.APIInfo().User.TenantID)
}
func (v *TOTenant) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"active":      dbhelpers.WhereColumnInfo{Column: "q.active", Checker: nil},
		"id":          dbhelpers.WhereColumnInfo{Column: "q.id", Checker: api.IsInt},
		"name":        dbhelpers.WhereColumnInfo{Column: "q.name", Checker: nil},
		"parent_id":   dbhelpers.WhereColumnInfo{Column: "q.parent_id", Checker: api.IsInt},
		"parent_name": dbhelpers.WhereColumnInfo{Column: "p.name", Checker: nil},
	}
}
func (v *TOTenant) UpdateQuery() string { return updateQuery() }

// GetID wraps the ID member with null checking
// Part of the Identifier interface
func (ten TOTenant) GetID() (int, bool) {
	if ten.ID == nil {
		return 0, false
	}
	return *ten.ID, true
}

// GetKeyFieldsInfo identifies types of the key fields
func (ten TOTenant) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

// GetKeys returns values of keys
func (ten TOTenant) GetKeys() (map[string]interface{}, bool) {
	var id int
	if ten.ID != nil {
		id = *ten.ID
	}
	return map[string]interface{}{"id": id}, true
}

// GetAuditName returns a unique identifier
// Part of the Identifier interface
func (ten TOTenant) GetAuditName() string {
	if ten.Name != nil {
		return *ten.Name
	}
	id, _ := ten.GetID()
	return strconv.Itoa(id)
}

// GetType returns the name of the type for messages
// Part of the Identifier interface
func (ten TOTenant) GetType() string {
	return "tenant"
}

// SetKeys allows CreateHandler to assign id once object is created.
// Part of the Identifier interface
func (ten *TOTenant) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	ten.ID = &i
}

// Validate fulfills the api.Validator interface
func (ten TOTenant) Validate() error {
	errs := validation.Errors{
		"name":       validation.Validate(ten.Name, validation.Required),
		"active":     validation.Validate(ten.Active), // only validate it's boolean
		"parentId":   validation.Validate(ten.ParentID, validation.Required, validation.Min(1)),
		"parentName": nil,
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (tn *TOTenant) Create() apierrors.Errors { return api.GenericCreate(tn) }

func (ten *TOTenant) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	if ten.APIInfo().User.TenantID == auth.TenantIDInvalid {
		return nil, apierrors.New(), nil
	}
	api.DefaultSort(ten.APIInfo(), "name")
	tenants, errs, maxTime := api.GenericRead(h, ten, useIMS)
	if errs.Occurred() {
		return nil, errs, nil
	}

	tenantNames := map[int]*string{}
	for _, it := range tenants {
		t := it.(*tc.TenantNullable)
		tenantNames[*t.ID] = t.Name
	}
	for _, it := range tenants {
		t := it.(*tc.TenantNullable)
		if t.ParentID == nil || tenantNames[*t.ParentID] == nil {
			// root tenant has no parent
			continue
		}
		p := *tenantNames[*t.ParentID]
		t.ParentName = &p // copy
	}
	return tenants, errs, maxTime
}

// IsTenantAuthorized implements the Tenantable interface for TOTenant
// returns true if the user has access on this tenant and on the ParentID if changed.
func (ten *TOTenant) IsTenantAuthorized(user *auth.CurrentUser) (bool, error) {
	var ok = false
	var err error

	if ten == nil {
		// should never happen
		return ok, err
	}

	if ten.ID != nil && *ten.ID != 0 {
		// modifying an existing tenant
		ok, err = tenant.IsResourceAuthorizedToUserTx(*ten.ID, user, ten.APIInfo().Tx.Tx)
		if !ok || err != nil {
			return ok, err
		}

		if ten.ParentID == nil || *ten.ParentID == 0 {
			// not changing parent
			return ok, err
		}

		// get current parentID to check if it's being changed
		var parentID int
		tx := ten.APIInfo().Tx.Tx
		err = tx.QueryRow(`SELECT parent_id FROM tenant WHERE id = ` + strconv.Itoa(*ten.ID)).Scan(&parentID)
		if err != nil {
			return false, err
		}
		if parentID == *ten.ParentID {
			// parent not being changed
			return ok, err
		}
	}

	// creating new tenant -- must specify a parent
	if ten.ParentID == nil || *ten.ParentID == 0 {
		return false, err
	}

	return tenant.IsResourceAuthorizedToUserTx(*ten.ParentID, user, ten.APIInfo().Tx.Tx)
}

func (tn *TOTenant) Update() apierrors.Errors { return api.GenericUpdate(tn) }

func (ten *TOTenant) Delete() apierrors.Errors {
	result, err := ten.APIInfo().Tx.NamedExec(deleteQuery(), ten)
	if err != nil {
		return parseDeleteErr(err, *ten.ID, ten.APIInfo().Tx.Tx) // this is why we can't use api.GenericDelete
	}

	errs := apierrors.New()
	if rowsAffected, err := result.RowsAffected(); err != nil {
		errs.SetSystemError("deleting " + ten.GetType() + ": getting rows affected: " + err.Error())
		errs.Code = http.StatusInternalServerError
	} else if rowsAffected < 1 {
		errs.SetUserError("no " + ten.GetType() + " with that id found")
		errs.Code = http.StatusNotFound
	} else if rowsAffected > 1 {
		errs.SystemError = fmt.Errorf(ten.GetType()+" delete affected too many rows: %d", rowsAffected)
		errs.Code = http.StatusInternalServerError
	}
	return errs
}

// parseDeleteErr takes the tenant delete error, and returns the appropriate user error, system error, and http status code.
func parseDeleteErr(err error, id int, tx *sql.Tx) apierrors.Errors {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return apierrors.Errors{
			SystemError: errors.New("deleting tenant: " + err.Error()),
			Code:        http.StatusInternalServerError,
		}
	}
	// TODO fix this to check for other Postgres errors besides key violations
	existing := ""
	switch pqErr.Table {
	case "tenant":
		existing = "child tenants"
	case "tm_user":
		existing = "users"
	case "deliveryservice":
		existing = "deliveryservices"
	case "origin":
		existing = "origins"
	default:
		existing = pqErr.Table
	}

	return apierrors.Errors{
		UserError: errors.New("Tenant '" + strconv.Itoa(id) + "' has " + existing + ". Please update these " + existing + " and retry."),
		Code:      http.StatusBadRequest,
	}
}

// selectQuery returns a query on the tenant table that limits to tenants within the realm of the tenantID.  It's intended
// to be extensible by adding WHERE/ORDERBY/etc clauses at the end to further refine the query.
func selectQuery(tenantID int) string {
	query := `
WITH RECURSIVE q AS (
SELECT id, name, active, parent_id, last_updated FROM tenant WHERE id = ` + strconv.Itoa(tenantID) + `
UNION SELECT t.id, t.name, t.active, t.parent_id, t.last_updated FROM tenant t JOIN q ON q.id = t.parent_id)
SELECT q.id AS id, q.name AS name, q.active AS active, q.parent_id AS parent_id, q.last_updated AS last_updated,
p.name AS parent_name FROM q LEFT JOIN tenant p ON q.parent_id = p.id
`

	return query
}

func updateQuery() string {
	query := `UPDATE
tenant SET
active=:active,
name=:name,
parent_id=:parent_id

WHERE id=:id RETURNING last_updated`
	return query
}

func insertQuery() string {
	query := `INSERT INTO tenant (
name,
active,
parent_id
) VALUES (
:name,
:active,
:parent_id
) RETURNING id,last_updated`
	return query
}

func deleteQuery() string {
	query := `DELETE FROM tenant
WHERE id=:id`
	return query
}
