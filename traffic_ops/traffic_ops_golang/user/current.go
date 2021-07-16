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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"
)

const replacePasswordQuery = `
UPDATE tm_user
SET local_passwd=$1
WHERE id=$2
`

const replaceConfirmPasswordQuery = `
UPDATE tm_user
SET confirm_local_passwd=$1
WHERE id=$2
`

const replaceCurrentQuery = `
UPDATE tm_user
SET address_line1=$1,
    address_line2=$2,
    city=$3,
    company=$4,
    country=$5,
    email=$6,
    full_name=$7,
    gid=$8,
    new_user=FALSE,
    phone_number=$9,
    postal_code=$10,
    public_ssh_key=$11,
    state_or_province=$12,
    tenant_id=$13,
    token=NULL,
    uid=$14,
    username=$15
WHERE id=$16
RETURNING address_line1,
          address_line2,
          city,
          company,
          country,
          email,
          full_name,
          gid,
          id,
          last_updated,
          new_user,
          phone_number,
          postal_code,
          public_ssh_key,
          role,
          (
          	SELECT role.name
          	FROM role
          	WHERE role.id=tm_user.role
          ) AS role_name,
          state_or_province,
          (
          	SELECT tenant.name
          	FROM tenant
          	WHERE tenant.id=tm_user.tenant_id
          ) AS tenant,
          tenant_id,
          uid,
          username
`

func Current(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	currentUser, err := getUser(inf.Tx.Tx, inf.User.ID)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("getting current user: "+err.Error()))
		return
	}

	version := inf.Version
	if version == nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, fmt.Errorf("TOUsers.Read called with invalid API version"), nil)
		return
	}
	if version.Major >= 4 {
		api.WriteResp(w, r, currentUser)
	} else {
		api.WriteResp(w, r, currentUser.UserCurrent)
	}
}

func getUser(tx *sql.Tx, id int) (tc.UserCurrentV40, error) {
	q := `
SELECT
u.address_line1,
u.address_line2,
u.city,
u.company,
u.country,
u.email,
u.full_name,
u.id,
u.last_updated,
u.last_authenticated,
u.local_passwd,
u.new_user,
u.phone_number,
u.postal_code,
u.public_ssh_key,
u.role,
r.name as role_name,
u.state_or_province,
t.name as tenant,
u.tenant_id,
u.username
FROM tm_user as u
LEFT JOIN role as r ON r.id = u.role
INNER JOIN tenant as t ON t.id = u.tenant_id
WHERE u.id=$1
`
	u := tc.UserCurrentV40{}
	localPassword := sql.NullString{}
	if err := tx.QueryRow(q, id).Scan(&u.AddressLine1, &u.AddressLine2, &u.City, &u.Company, &u.Country, &u.Email, &u.FullName, &u.ID, &u.LastUpdated, &u.LastAuthenticated, &localPassword, &u.NewUser, &u.PhoneNumber, &u.PostalCode, &u.PublicSSHKey, &u.Role, &u.RoleName, &u.StateOrProvince, &u.Tenant, &u.TenantID, &u.UserName); err != nil {
		return tc.UserCurrentV40{}, errors.New("querying current user: " + err.Error())
	}
	u.LocalUser = util.BoolPtr(localPassword.Valid)
	return u, nil
}

func ReplaceCurrent(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	tx := inf.Tx.Tx
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	var userRequest tc.CurrentUserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&userRequest); err != nil {
		errCode = http.StatusBadRequest
		userErr = fmt.Errorf("Couldn't parse request: %v", err)
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	user, exists, err := dbhelpers.GetUserByID(inf.User.ID, tx)
	if err != nil {
		sysErr = fmt.Errorf("Getting user by ID %d: %v", inf.User.ID, err)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}
	if !exists {
		sysErr = fmt.Errorf("Current user (#%d) doesn't exist... ??", inf.User.ID)
		errCode = http.StatusInternalServerError
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	if err := userRequest.User.UnmarshalAndValidate(&user); err != nil {
		errCode = http.StatusBadRequest
		userErr = fmt.Errorf("Couldn't parse request: %v", err)
		api.HandleErr(w, r, tx, errCode, userErr, nil)
		return
	}

	changePasswd := false
	changeConfirmPasswd := false

	// obfuscate passwords (UnmarshalAndValidate checks for equality with ConfirmLocalPassword)
	// TODO: check for valid password via bad password list like Perl did? User creation doesn't...
	if user.LocalPassword != nil && *user.LocalPassword != "" {
		if ok, err := auth.IsGoodPassword(*user.LocalPassword); !ok {
			errCode = http.StatusBadRequest
			if err != nil {
				userErr = err
			} else {
				userErr = fmt.Errorf("Unacceptable password")
			}
			api.HandleErr(w, r, tx, errCode, userErr, nil)
			return
		}

		hashPass, err := auth.DerivePassword(*user.LocalPassword)
		if err != nil {
			sysErr = fmt.Errorf("Hashing new password: %v", err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		}
		changePasswd = true
		user.LocalPassword = util.StrPtr(hashPass)
	}

	// Perl did this although it serves no known purpose
	if user.ConfirmLocalPassword != nil && *user.ConfirmLocalPassword != "" {
		hashPass, err := auth.DerivePassword(*user.ConfirmLocalPassword)
		if err != nil {
			sysErr = fmt.Errorf("Hashing new 'confirm' password: %v", err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		}
		user.ConfirmLocalPassword = util.StrPtr(hashPass)
		changeConfirmPasswd = true
	}

	if *user.Role != inf.User.Role {
		privLevel, exists, err := dbhelpers.GetPrivLevelFromRoleID(tx, *user.Role)
		if err != nil {
			sysErr = fmt.Errorf("Getting privLevel for Role #%d: %v", *user.Role, err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		}
		if !exists {
			userErr = fmt.Errorf("role: no such role: %d", *user.Role)
			errCode = http.StatusNotFound
			api.HandleErr(w, r, tx, errCode, userErr, nil)
			return
		}
		if privLevel > inf.User.PrivLevel {
			userErr = errors.New("role: cannot have greater permissions than user's current role")
			errCode = http.StatusForbidden
			api.HandleErr(w, r, tx, errCode, userErr, nil)
			return
		}
	}

	if ok, err := tenant.IsResourceAuthorizedToUserTx(*user.TenantID, inf.User, tx); err != nil {
		if err == sql.ErrNoRows {
			userErr = errors.New("No such tenant!")
			errCode = http.StatusNotFound
		} else {
			sysErr = fmt.Errorf("Checking user %s permissions on tenant #%d: %v", inf.User.UserName, *user.TenantID, err)
			errCode = http.StatusInternalServerError
		}
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	} else if !ok {
		// unlike Perl, this endpoint will not disclose the existence of tenants over which the current
		// user has no permission - in keeping with the behavior of the '/tenants' endpoint.
		userErr = errors.New("No such tenant!")
		errCode = http.StatusNotFound
		api.HandleErr(w, r, tx, errCode, userErr, sysErr)
		return
	}

	if *user.Username != inf.User.UserName {

		if ok, err := dbhelpers.UsernameExists(*user.Username, tx); err != nil {
			sysErr = fmt.Errorf("Checking existence of user %s: %v", *user.Username, err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		} else if ok {
			// TODO users are tenanted, so theoretically I should be hiding the existence of the
			// conflicting user - but then how do I tell the client how to fix their request?
			userErr = fmt.Errorf("Username %s already exists!", *user.Username)
			errCode = http.StatusConflict
			api.HandleErr(w, r, tx, errCode, userErr, nil)
			return
		}
	}

	if err = updateUser(&user, tx, changePasswd, changeConfirmPasswd); err != nil {
		errCode = http.StatusInternalServerError
		sysErr = fmt.Errorf("Updating user: %v", err)
		api.HandleErr(w, r, tx, errCode, nil, sysErr)
		return
	}

	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "User profile was successfully updated", user)
}

func updateUser(u *tc.User, tx *sql.Tx, changePassword bool, changeConfirmPasswd bool) error {
	row := tx.QueryRow(replaceCurrentQuery,
		u.AddressLine1,
		u.AddressLine2,
		u.City,
		u.Company,
		u.Country,
		u.Email,
		u.FullName,
		u.GID,
		u.PhoneNumber,
		u.PostalCode,
		u.PublicSSHKey,
		u.StateOrProvince,
		u.TenantID,
		u.UID,
		u.Username,
		u.ID,
	)

	err := row.Scan(&u.AddressLine1,
		&u.AddressLine2,
		&u.City,
		&u.Company,
		&u.Country,
		&u.Email,
		&u.FullName,
		&u.GID,
		&u.ID,
		&u.LastUpdated,
		&u.NewUser,
		&u.PhoneNumber,
		&u.PostalCode,
		&u.PublicSSHKey,
		&u.Role,
		&u.RoleName,
		&u.StateOrProvince,
		&u.Tenant,
		&u.TenantID,
		&u.UID,
		&u.Username,
	)
	if err != nil {
		return err
	}

	if changePassword {
		_, err = tx.Exec(replacePasswordQuery, u.LocalPassword, u.ID)
		if err != nil {
			return fmt.Errorf("resetting password: %v", err)
		}
	}

	if changeConfirmPasswd {
		_, err = tx.Exec(replaceConfirmPasswordQuery, u.ConfirmLocalPassword, u.ID)
		if err != nil {
			return fmt.Errorf("resetting confirm password: %v", err)
		}
	}

	u.LocalPassword = nil
	u.ConfirmLocalPassword = nil
	return nil
}
