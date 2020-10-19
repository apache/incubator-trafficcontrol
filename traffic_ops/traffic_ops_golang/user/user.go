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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/util/ims"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

type TOUser struct {
	api.APIInfoImpl `json:"-"`
	tc.User
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

func (user *TOUser) SetLastUpdated(t tc.TimeNoMod) {
	user.LastUpdated = &t
}

func (user *TOUser) NewReadObj() interface{} {
	return &tc.User{}
}

func (user *TOUser) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"id":       dbhelpers.WhereColumnInfo{"u.id", api.IsInt},
		"role":     dbhelpers.WhereColumnInfo{"r.name", nil},
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
func (user *TOUser) Create() apierrors.Errors {

	// PUT and POST validation differs slightly
	err := user.postValidate()
	if err != nil {
		return apierrors.Errors{
			Code:      http.StatusBadRequest,
			UserError: err,
		}
	}

	// make sure the user cannot create someone with a higher priv_level than themselves
	if errs := user.privCheck(); errs.Occurred() {
		return errs
	}

	// Convert password to SCRYPT
	*user.LocalPassword, err = auth.DerivePassword(*user.LocalPassword)
	if err != nil {
		return apierrors.Errors{
			Code:      http.StatusBadRequest,
			UserError: err,
		}
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

	errs := apierrors.Errors{
		Code: http.StatusInternalServerError,
	}
	rowsAffected := 0
	for resultRows.Next() {
		rowsAffected++
		if err = resultRows.Scan(&id, &lastUpdated, &tenant, &rolename); err != nil {
			errs.SystemError = fmt.Errorf("could not scan after insert: %v)", err)
			return errs
		}
	}

	if rowsAffected == 0 {
		errs.SetSystemError("no user was inserted, nothing was returned")
		return errs
	} else if rowsAffected > 1 {
		errs.SetSystemError("too many rows affected from user insert")
		return errs
	}

	user.ID = &id
	user.LastUpdated = &lastUpdated
	user.Tenant = &tenant
	user.RoleName = &rolename
	user.LocalPassword = nil

	return apierrors.New()
}

// This is not using GenericRead because of this tenancy check. Maybe we can add tenancy functionality to the generic case?
func (this *TOUser) Read(h http.Header, useIMS bool) ([]interface{}, apierrors.Errors, *time.Time) {
	var maxTime time.Time
	var runSecond bool
	inf := this.APIInfo()
	api.DefaultSort(inf, "username")
	errs := apierrors.New()
	where, orderBy, pagination, queryValues, dbErrs := dbhelpers.BuildWhereAndOrderByAndPagination(inf.Params, this.ParamColumns())
	if len(dbErrs) > 0 {
		errs.UserError = util.JoinErrs(dbErrs)
		errs.Code = http.StatusBadRequest
		return nil, errs, nil
	}

	tenantIDs, err := tenant.GetUserTenantIDListTx(inf.Tx.Tx, inf.User.TenantID)
	if err != nil {
		errs.SystemError = fmt.Errorf("getting tenant list for user: %v", err)
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	where, queryValues = dbhelpers.AddTenancyCheck(where, queryValues, "u.tenant_id", tenantIDs)

	if useIMS {
		runSecond, maxTime = ims.TryIfModifiedSinceQuery(this.APIInfo().Tx, h, queryValues, selectMaxLastUpdatedQuery(where))
		if !runSecond {
			log.Debugln("IMS HIT")
			return []interface{}{}, apierrors.Errors{Code: http.StatusNotModified}, &maxTime
		}
		log.Debugln("IMS MISS")
	} else {
		log.Debugln("Non IMS request")
	}
	query := this.SelectQuery() + where + orderBy + pagination
	rows, err := inf.Tx.NamedQuery(query, queryValues)
	if err != nil {
		errs.SystemError = fmt.Errorf("querying users : %v", err)
		errs.Code = http.StatusInternalServerError
		return nil, errs, nil
	}
	defer rows.Close()

	type UserGet struct {
		RoleName *string `json:"rolename" db:"rolename"`
		tc.User
	}

	user := &UserGet{}
	users := []interface{}{}
	for rows.Next() {
		if err = rows.StructScan(user); err != nil {
			errs.SystemError = fmt.Errorf("parsing user rows: %v", err)
			errs.Code = http.StatusInternalServerError
			return nil, errs, nil
		}
		users = append(users, *user)
	}

	return users, errs, &maxTime
}

func selectMaxLastUpdatedQuery(where string) string {
	return `SELECT max(t) from (
		SELECT max(u.last_updated) as t FROM tm_user u
		LEFT JOIN tenant t ON u.tenant_id = t.id
		LEFT JOIN role r ON u.role = r.id ` + where +
		` UNION ALL
	select max(last_updated) as t from last_deleted l where l.table_name='tm_user') as res`
}

func (user *TOUser) privCheck() apierrors.Errors {
	requestedPrivLevel, _, err := dbhelpers.GetPrivLevelFromRoleID(user.ReqInfo.Tx.Tx, *user.Role)
	if err != nil {
		return apierrors.Errors{
			Code:        http.StatusInternalServerError,
			SystemError: err,
		}
	}

	if user.ReqInfo.User.PrivLevel < requestedPrivLevel {
		return apierrors.Errors{
			Code:      http.StatusForbidden,
			UserError: errors.New("user cannot update a user with a role more privileged than themselves"),
		}
	}

	return apierrors.New()
}

func (user *TOUser) Update() apierrors.Errors {

	// make sure current user cannot update their own role to a new value
	if user.ReqInfo.User.ID == *user.ID && user.ReqInfo.User.Role != *user.Role {
		return apierrors.Errors{
			Code:      http.StatusBadRequest,
			UserError: fmt.Errorf("users cannot update their own role"),
		}
	}

	// make sure the user cannot update someone with a higher priv_level than themselves
	errs := user.privCheck()
	if errs.Occurred() {
		return errs
	}

	if user.LocalPassword != nil {
		var err error
		*user.LocalPassword, err = auth.DerivePassword(*user.LocalPassword)
		if err != nil {
			errs.SystemError = err
			errs.Code = http.StatusInternalServerError
			return errs
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
			errs.SystemError = fmt.Errorf("could not scan lastUpdated from insert: %s", err)
			errs.Code = http.StatusInternalServerError
			return errs
		}
	}

	user.LastUpdated = &lastUpdated
	user.Tenant = &tenant
	user.RoleName = &rolename
	user.LocalPassword = nil

	if rowsAffected != 1 {
		if rowsAffected < 1 {
			errs.SetUserError("no user found with this id")
			errs.Code = http.StatusNotFound
		} else {
			errs.SystemError = fmt.Errorf("this update affected too many rows: %d", rowsAffected)
			errs.Code = http.StatusInternalServerError
		}
	}

	return errs
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
