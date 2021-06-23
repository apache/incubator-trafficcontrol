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
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/lib/pq"
)

const rootName = `root`

// TOTenant provides a local type against which to define methods
type TOTenant struct {
	api.InfoerImpl `json:"-"`
	tc.TenantNullable
}

func (ten *TOTenant) GetLastUpdated() (*time.Time, bool, error) {
	return api.GetLastUpdated(ten.Info().Tx, *ten.ID, "tenant")
}

func (ten *TOTenant) SetLastUpdated(t tc.TimeNoMod) { ten.LastUpdated = &t }
func (ten *TOTenant) InsertQuery() string           { return insertQuery() }
func (ten *TOTenant) SelectMaxLastUpdatedQuery(where, orderBy, pagination, tableName string) string {
	return `SELECT max(t) from (
		SELECT max(last_updated) as t from ` + tableName + ` q ` + where + orderBy + pagination +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='` + tableName + `') as res`
}
func (ten *TOTenant) NewReadObj() interface{} { return &tc.TenantNullable{} }
func (ten *TOTenant) SelectQuery() string {
	return selectQuery(ten.Info().User.TenantID)
}
func (ten *TOTenant) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"active":      dbhelpers.WhereColumnInfo{Column: "q.active", Checker: nil},
		"id":          dbhelpers.WhereColumnInfo{Column: "q.id", Checker: api.IsInt},
		"name":        dbhelpers.WhereColumnInfo{Column: "q.name", Checker: nil},
		"parent_id":   dbhelpers.WhereColumnInfo{Column: "q.parent_id", Checker: api.IsInt},
		"parent_name": dbhelpers.WhereColumnInfo{Column: "p.name", Checker: nil},
	}
}
func (ten *TOTenant) UpdateQuery() string { return updateQuery() }

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

func (ten *TOTenant) Create() (error, error, int) { return api.GenericCreate(ten) }

func (ten *TOTenant) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	if ten.Info().User.TenantID == auth.TenantIDInvalid {
		return nil, nil, nil, http.StatusOK, nil
	}
	api.DefaultSort(ten.Info(), "name")
	tenants, userErr, sysErr, errCode, maxTime := api.GenericRead(h, ten, useIMS)
	if userErr != nil || sysErr != nil {
		return nil, userErr, sysErr, errCode, nil
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
	return tenants, nil, nil, errCode, maxTime
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
		ok, err = tenant.IsResourceAuthorizedToUserTx(*ten.ID, user, ten.Info().Tx.Tx)
		if !ok || err != nil {
			return ok, err
		}

		if ten.ParentID == nil || *ten.ParentID == 0 {
			// not changing parent
			return ok, err
		}

		// get current parentID to check if it's being changed
		var parentID int
		tx := ten.Info().Tx.Tx
		// If it's the root tenant, don't check for parent
		if ten.Name != nil && *ten.Name != rootName {
			err = tx.QueryRow(`SELECT parent_id FROM tenant WHERE id = ` + strconv.Itoa(*ten.ID)).Scan(&parentID)
			if err != nil {
				return false, err
			}
			if parentID == *ten.ParentID {
				// parent not being changed
				return ok, err
			}
		}
	}

	// creating new tenant -- must specify a parent
	if ten.ParentID == nil || *ten.ParentID == 0 {
		return false, err
	}

	return tenant.IsResourceAuthorizedToUserTx(*ten.ParentID, user, ten.Info().Tx.Tx)
}

// Update wraps tenant validation and the generic API Update call into a single call.
func (ten *TOTenant) Update(h http.Header) (error, error, int) {

	userErr, sysErr, statusCode := ten.isUpdatable()
	if userErr != nil || sysErr != nil {
		return userErr, sysErr, statusCode
	}

	return api.GenericUpdate(h, ten)
}

// isUpdatable peforms validation on the fields for the Tenant, such as ensuring
// the tenant cannot be modified if it is root, or that it cannot convert its own child
// to its own parent. This is different than the basic validation rules performed in
// Validate() as it pertains to specific business logic, not generic API rules.
func (ten *TOTenant) isUpdatable() (error, error, int) {
	if ten.Name != nil && *ten.Name == rootName {
		return errors.New("trying to change the root tenant, which is immutable"), nil, http.StatusBadRequest
	}

	// Perform SelectQuery
	vals := []tc.TenantNullable{}
	query := selectQuery(*ten.ID)
	rows, err := ten.Info().Tx.Queryx(query)
	if err != nil {
		return nil, errors.New("querying " + ten.GetType() + ": " + err.Error()), http.StatusInternalServerError
	}
	defer rows.Close()

	for rows.Next() {
		var v tc.TenantNullable
		if err = rows.StructScan(&v); err != nil {
			return nil, errors.New("scanning " + ten.GetType() + ": " + err.Error()), http.StatusInternalServerError
		}
		vals = append(vals, v)
	}

	// Ensure the new desired ParentID does not exist in the susequent list of Children
	for _, val := range vals {
		if *ten.ParentID == *val.ID {
			return errors.New("trying to set existing child as new parent"), nil, http.StatusBadRequest
		}
	}
	return nil, nil, http.StatusOK
}

func (ten *TOTenant) Delete() (error, error, int) {
	result, err := ten.Info().Tx.NamedExec(deleteQuery(), ten)
	if err != nil {
		return parseDeleteErr(err, *ten.ID, ten.Info().Tx.Tx) // this is why we can't use api.GenericDelete
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return nil, errors.New("deleting " + ten.GetType() + ": getting rows affected: " + err.Error()), http.StatusInternalServerError
	} else if rowsAffected < 1 {
		return errors.New("no " + ten.GetType() + " with that id found"), nil, http.StatusNotFound
	} else if rowsAffected > 1 {
		return nil, fmt.Errorf(ten.GetType()+" delete affected too many rows: %d", rowsAffected), http.StatusInternalServerError
	}
	return nil, nil, http.StatusOK
}

// parseDeleteErr takes the tenant delete error, and returns the appropriate user error, system error, and http status code.
func parseDeleteErr(err error, id int, tx *sql.Tx) (error, error, int) {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return nil, errors.New("deleting tenant: " + err.Error()), http.StatusInternalServerError
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
	return errors.New("Tenant '" + strconv.Itoa(id) + "' has " + existing + ". Please update these " + existing + " and retry."), nil, http.StatusBadRequest
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
