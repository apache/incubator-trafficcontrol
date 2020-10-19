package auth

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
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/apierrors"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type CurrentUser struct {
	UserName     string         `json:"userName" db:"username"`
	ID           int            `json:"id" db:"id"`
	PrivLevel    int            `json:"privLevel" db:"priv_level"`
	TenantID     int            `json:"tenantId" db:"tenant_id"`
	Role         int            `json:"role" db:"role"`
	Capabilities pq.StringArray `json:"capabilities" db:"capabilities"`
}

type PasswordForm struct {
	Username string `json:"u"`
	Password string `json:"p"`
}

const disallowed = "disallowed"

// PrivLevelInvalid - The Default Priv level
const PrivLevelInvalid = -1

const PrivLevelReadOnly = 10

const PrivLevelORT = 11

const PrivLevelSteering = 15

const PrivLevelFederation = 15

const PrivLevelPortal = 15

const PrivLevelOperations = 20

const PrivLevelAdmin = 30

// TenantIDInvalid - The default Tenant ID
const TenantIDInvalid = -1

type key int

const CurrentUserKey key = iota

// GetCurrentUserFromDB returns the id and privilege level of the given user
// along with the username, or -1 as the id, - as the userName and
// PrivLevelInvalid if the user doesn't exist, along with an error code to
// return, a user-facing error, and a system error to log.
func GetCurrentUserFromDB(DB *sqlx.DB, user string, timeout time.Duration) (CurrentUser, apierrors.Errors) {
	qry := `
SELECT
  r.priv_level,
  r.id as role,
  u.id,
  u.username,
  COALESCE(u.tenant_id, -1) AS tenant_id,
  ARRAY(SELECT rc.cap_name FROM role_capability AS rc WHERE rc.role_id=r.id) AS capabilities
FROM
  tm_user AS u
JOIN
  role AS r ON u.role = r.id
WHERE
  u.username = $1
`

	var u CurrentUser

	errs := apierrors.New()
	if DB == nil {
		errs.SetSystemError("no db provided to GetCurrentUserFromDB")
		errs.Code = http.StatusInternalServerError
		return u, errs
	}
	dbCtx, dbClose := context.WithTimeout(context.Background(), timeout)
	defer dbClose()

	err := DB.GetContext(dbCtx, &u, qry, user)
	switch {
	case err == sql.ErrNoRows:
		errs.SetUserError("user not found")
		errs.SystemError = fmt.Errorf("checking user %v info: user not in database", user)
		errs.Code = http.StatusUnauthorized
	case err == context.DeadlineExceeded || err == context.Canceled:
		errs.SystemError = fmt.Errorf("db access timed out: %s number of open connections: %d", err, DB.Stats().OpenConnections)
		errs.Code = http.StatusInternalServerError
	case err != nil:
		errs.SystemError = fmt.Errorf("error checking user %s info: %v", user, err)
		errs.Code = http.StatusInternalServerError
	}
	return u, errs
}

func GetCurrentUser(ctx context.Context) (*CurrentUser, error) {
	val := ctx.Value(CurrentUserKey)
	if val != nil {
		switch v := val.(type) {
		case CurrentUser:
			return &v, nil
		default:
			return nil, fmt.Errorf("CurrentUser found with bad type: %T", v)
		}
	}
	return &CurrentUser{"-", -1, PrivLevelInvalid, TenantIDInvalid, -1, []string{}}, errors.New("No user found in Context")
}

func CheckLocalUserIsAllowed(form PasswordForm, db *sqlx.DB, timeout time.Duration) (bool, error, error) {
	var roleName string
	dbCtx, dbClose := context.WithTimeout(context.Background(), timeout)
	defer dbClose()

	err := db.GetContext(dbCtx, &roleName, "SELECT role.name FROM role INNER JOIN tm_user ON tm_user.role = role.id where username=$1", form.Username)
	if err != nil {
		if err == context.DeadlineExceeded || err == context.Canceled {
			return false, nil, err
		}
		return false, err, nil
	}
	if roleName != "" {
		if roleName != disallowed { //relies on unchanging role name assumption.
			return true, nil, nil
		}
	}
	return false, nil, nil
}

func CheckLocalUserPassword(form PasswordForm, db *sqlx.DB, timeout time.Duration) (bool, error, error) {
	var hashedPassword string
	dbCtx, dbClose := context.WithTimeout(context.Background(), timeout)
	defer dbClose()

	err := db.GetContext(dbCtx, &hashedPassword, "SELECT local_passwd FROM tm_user WHERE username=$1", form.Username)
	if err != nil {
		if err == context.DeadlineExceeded || err == context.Canceled {
			return false, nil, err
		}
		return false, err, nil
	}
	err = VerifySCRYPTPassword(form.Password, hashedPassword)
	if err != nil {
		hashedInput, err := sha1Hex(form.Password)
		if err != nil {
			return false, err, nil
		}
		if hashedPassword == hashedInput { // for backwards compatibility
			return true, nil, nil
		}
		return false, err, nil
	}
	return true, nil, nil
}

// CheckLocalUserToken checks the passed token against the records in the db for a match, up to a
// maximum duration of timeout.
func CheckLocalUserToken(token string, db *sqlx.DB, timeout time.Duration) (bool, string, error) {
	dbCtx, dbClose := context.WithTimeout(context.Background(), timeout)
	defer dbClose()

	var username string
	err := db.GetContext(dbCtx, &username, `SELECT username FROM tm_user WHERE token=$1 AND role!=(SELECT role.id FROM role WHERE role.name=$2)`, token, disallowed)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	return true, username, nil
}

func sha1Hex(s string) (string, error) {
	// SHA1 hash
	hash := sha1.New()
	if _, err := hash.Write([]byte(s)); err != nil {
		return "", err
	}
	hashBytes := hash.Sum(nil)
	hexSha1 := hex.EncodeToString(hashBytes)
	return hexSha1, nil
}

func CheckLDAPUser(form PasswordForm, cfg *config.ConfigLDAP) (bool, error) {
	userDN, valid, err := LookupUserDN(form.Username, cfg)
	if err != nil {
		return false, err
	}
	if valid {
		return AuthenticateUserDN(userDN, form.Password, cfg)
	}
	return false, errors.New("User not found in LDAP")
}
