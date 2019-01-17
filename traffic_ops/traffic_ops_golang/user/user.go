package user

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

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

type TOUser struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.User
}

func GetTypeSingleton() api.CRUDFactory {
	return func(reqInfo *api.APIInfo) api.CRUDer {
		toReturn := TOUser{reqInfo, tc.User{}}
		return &toReturn
	}
}

func (user TOUser) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{Field: "id", Func: api.GetIntKey}}
}

func (user TOUser) GetKeys() (map[string]interface{}, bool) {
	if user.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *user.ID}, true
}

func (user TOUser) GetAuditName() string {
	if user.Username != nil {
		return *user.Username
	}
	if user.ID != nil {
		return strconv.Itoa(*user.ID)
	}
	return "unknown"
}

func (user TOUser) GetType() string {
	return "user"
}

func (user *TOUser) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) // non-panicking type assertion
	user.ID = &i
}

func (v *TOUser) APIInfo() *api.APIInfo {
	return v.ReqInfo
}

func (user *TOUser) SetLastUpdated(t tc.TimeNoMod) {
	user.LastUpdated = &t
}

func (user *TOUser) NewReadObj() interface{} {
	return &tc.User{}
}

func (user *TOUser) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"id":       dbhelpers.WhereColumnInfo{"u.id", api.IsInt},
		"tenant":   dbhelpers.WhereColumnInfo{"t.name", nil},
		"username": dbhelpers.WhereColumnInfo{"u.username", nil},
	}
}

func (user *TOUser) Validate() error {

	validateErrs := validation.Errors{
		"email":    validation.Validate(user.Email, validation.Required, is.Email),
		"fullName": validation.Validate(user.FullName, validation.Required),
		"role":     validation.Validate(user.Role, validation.Required),
		"username": validation.Validate(user.Username, validation.Required),
		"tenantID": validation.Validate(user.TenantID, validation.Required),
	}

	// Password is not required for update
	if user.LocalPassword != nil {
		_, err := auth.IsGoodLoginPair(*user.Username, *user.LocalPassword)
		if err != nil {
			return err
		}
	}

	return util.JoinErrs(tovalidate.ToErrors(validateErrs))
}

func (user *TOUser) postValidate() error {
	validateErrs := validation.Errors{
		"localPasswd": validation.Validate(user.LocalPassword, validation.Required),
	}
	return util.JoinErrs(tovalidate.ToErrors(validateErrs))
}

// Note: Not using GenericCreate because Scan also needs to scan tenant and rolename
func (user *TOUser) Create() (error, error, int) {

	// PUT and POST validation differs slightly
	err := user.postValidate()
	if err != nil {
		return err, nil, http.StatusBadRequest
	}

	if usrErr, sysErr, code := user.privCheck(); code != http.StatusOK {
		return usrErr, sysErr, code
	}

	// Convert password to SCRYPT
	*user.LocalPassword, err = auth.DerivePassword(*user.LocalPassword)
	if err != nil {
		return err, nil, http.StatusBadRequest
	}

	resultRows, err := user.ReqInfo.Tx.NamedQuery(user.InsertQuery(), user)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer resultRows.Close()

	var id int
	var lastUpdated tc.TimeNoMod
	var tenant string
	var rolename string

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err = resultRows.Scan(&id, &lastUpdated, &tenant, &rolename); err != nil {
			return nil, fmt.Errorf("could not scan after insert: %s\n)", err), http.StatusInternalServerError
		}
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("no user was inserted, nothing was returned"), http.StatusInternalServerError
	} else if rowsAffected > 1 {
		return nil, fmt.Errorf("too many rows affected from user insert"), http.StatusInternalServerError
	}

	user.ID = &id
	user.LastUpdated = &lastUpdated
	user.Tenant = &tenant
	user.RoleName = &rolename
	user.LocalPassword = nil

	return nil, nil, http.StatusOK
}

// returning true indicates the data related to the given tenantID should be visible
// this is just a linear search;`tenantIDs` is presumed to be unsorted
func checkTenancy(tenantID *int, tenantIDs []int) bool {
	for _, id := range tenantIDs {
		if id == *tenantID {
			return true
		}
	}
	return false
}

// This is not using GenericRead because of this tenancy check. Maybe we can add tenancy functionality to the generic case?
func (user *TOUser) Read() ([]interface{}, error, error, int) {

	var query string
	where, orderBy, queryValues, errs := dbhelpers.BuildWhereAndOrderBy(user.APIInfo().Params, user.ParamColumns())

	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest
	}

	tenantIDs, err := tenant.GetUserTenantIDListTx(user.ReqInfo.Tx.Tx, user.ReqInfo.User.TenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting tenant list for user: %v\n", err), http.StatusInternalServerError
	}
	where, queryValues = dbhelpers.AddTenancyCheck(where, queryValues, "u.tenant_id", tenantIDs)

	query = user.SelectQuery() + where + orderBy
	rows, err := user.ReqInfo.Tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, nil, fmt.Errorf("querying users : %v", err), http.StatusInternalServerError
	}
	defer rows.Close()

	users := []interface{}{}
	for rows.Next() {

		if err = rows.StructScan(user); err != nil {
			return nil, nil, fmt.Errorf("parsing user rows: %v", err), http.StatusInternalServerError
		}

		// role is a required field for the endpoint, but not for an item in the database
		// I doubt a nil check is needed, but I'm including it just in case
		if user.RoleName == nil {
			return nil, nil, fmt.Errorf("role name is nil", err), http.StatusInternalServerError
		}
		user.RoleNameGET = user.RoleName
		user.RoleName = nil

		users = append(users, *user)
	}

	return users, nil, nil, http.StatusOK
}

func (user *TOUser) privCheck() (error, error, int) {
	privLevel, _, err := dbhelpers.GetPrivLevelFromRoleID(user.ReqInfo.Tx, *user.Role)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	// what is the original error?
	if user.ReqInfo.User.PrivLevel < privLevel {
		return fmt.Errorf("user cannot update a user with a role more privileged than themselves"), nil, http.StatusForbidden
	}

	return nil, nil, http.StatusOK
}

func (user *TOUser) Update() (error, error, int) {

	// user is updating themselves
	if user.ReqInfo.User.ID == *user.ID {
		if user.ReqInfo.User.Role != *user.Role {
			return fmt.Errorf("users cannot update their own role"), nil, http.StatusBadRequest
		}
	}

	if usrErr, sysErr, code := user.privCheck(); code != http.StatusOK {
		return usrErr, sysErr, code
	}

	if user.LocalPassword != nil {
		var err error
		*user.LocalPassword, err = auth.DerivePassword(*user.LocalPassword)
		if err != nil {
			return nil, err, http.StatusInternalServerError
		}
	}

	resultRows, err := user.ReqInfo.Tx.NamedQuery(user.UpdateQuery(), user)
	if err != nil {
		return api.ParseDBError(err)
	}
	defer resultRows.Close()

	var lastUpdated tc.TimeNoMod
	var tenant string
	var rolename string

	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err := resultRows.Scan(&lastUpdated, &tenant, &rolename); err != nil {
			return nil, fmt.Errorf("could not scan lastUpdated from insert: %s\n", err), http.StatusInternalServerError
		}
	}

	user.LastUpdated = &lastUpdated
	user.Tenant = &tenant
	user.RoleName = &rolename
	user.LocalPassword = nil

	if rowsAffected != 1 {
		if rowsAffected < 1 {
			return fmt.Errorf("no user found with this id"), nil, http.StatusNotFound
		}
		return nil, fmt.Errorf("this update affected too many rows: %d", rowsAffected), http.StatusInternalServerError
	}

	return nil, nil, http.StatusOK
}

// Delete is unimplemented, needed to satisfy CRUDer
func (user *TOUser) Delete() (error, error, int) {
	return nil, nil, http.StatusNotImplemented
}

func (u *TOUser) IsTenantAuthorized(user *auth.CurrentUser) (bool, error) {

	// Delete: only id is given
	// Create: only tenant id
	// Update: id and tenant id
	//	id is associated with old tenant id
	//	we need to also check new tenant id

	tx := u.ReqInfo.Tx.Tx

	if u.ID != nil { // old tenant id (only on update or delete)

		var tenantID int
		if err := tx.QueryRow(`SELECT tenant_id from tm_user WHERE id = $1`, *u.ID).Scan(&tenantID); err != nil {
			if err != sql.ErrNoRows {
				return false, err
			}

			// At this point, tenancy isn't technically 'true', but I can't return a resource not found error here.
			// Letting it continue will let it run into a 404 when it tries to update.
			return true, nil
		}

		//log.Debugf("%d with tenancy %d trying to access %d with tenancy %d", user.ID, user.TenantID, *u.ID, tenantID)
		authorized, err := tenant.IsResourceAuthorizedToUserTx(tenantID, user, tx)
		if err != nil {
			return false, err
		}
		if !authorized {
			return false, nil

		}
	}

	if u.TenantID != nil { // new tenant id (only on create or udpate)

		//log.Debugf("%d with tenancy %d trying to access %d", user.ID, user.TenantID, *u.TenantID)
		authorized, err := tenant.IsResourceAuthorizedToUserTx(*u.TenantID, user, tx)
		if err != nil {
			return false, err
		}
		if !authorized {
			return false, nil
		}
	}

	return true, nil
}

func (user *TOUser) SelectQuery() string {
	return `
	SELECT
	u.id,
	u.username as username,
	u.public_ssh_key,
	u.role,
	r.name as rolename,
	u.company,
	u.email,
	u.full_name,
	u.new_user,
	u.address_line1,
	u.address_line2,
	u.city,
	u.state_or_province,
	u.phone_number,
	u.postal_code,
	u.country,
	u.registration_sent,
	u.tenant_id,
	t.name as tenant,
	u.last_updated
	FROM tm_user u
	LEFT JOIN tenant t ON u.tenant_id = t.id
	LEFT JOIN role r ON u.role = r.id`
}

func (user *TOUser) UpdateQuery() string {
	return `
	UPDATE tm_user u SET
	username=:username,
	public_ssh_key=:public_ssh_key,
	role=:role,
	company=:company,
	email=:email,
	full_name=:full_name,
	new_user=COALESCE(:new_user, FALSE),
	address_line1=:address_line1,
	address_line2=:address_line2,
	city=:city,
	state_or_province=:state_or_province,
	phone_number=:phone_number,
	postal_code=:postal_code,
	country=:country,
	registration_sent=:registration_sent,
	tenant_id=:tenant_id,
	local_passwd=COALESCE(:local_passwd, local_passwd)
	WHERE id=:id
	RETURNING last_updated,
	 (SELECT t.name FROM tenant t WHERE id = u.tenant_id),
	 (SELECT r.name FROM role r WHERE id = u.role)`
}

func (user *TOUser) InsertQuery() string {
	return `
	INSERT INTO tm_user (
	username,
	public_ssh_key,
	role,
	company,
	email,
	full_name,
	new_user,
	address_line1,
	address_line2,
	city,
	state_or_province,
	phone_number,
	postal_code,
	country,
	tenant_id,
	local_passwd
	) VALUES (
	:username,
	:public_ssh_key,
	:role,
	:company,
	:email,
	:full_name,
	COALESCE(:new_user, FALSE),
	:address_line1,
	:address_line2,
	:city,
	:state_or_province,
	:phone_number,
	:postal_code,
	:country,
	:tenant_id,
	:local_passwd
	) RETURNING id, last_updated,
	(SELECT t.name FROM tenant t WHERE id = tm_user.tenant_id),
	(SELECT r.name FROM role r WHERE id = tm_user.role)`
}

func (user *TOUser) DeleteQuery() string {
	return `DELETE FROM tm_user WHERE id = :id`
}
