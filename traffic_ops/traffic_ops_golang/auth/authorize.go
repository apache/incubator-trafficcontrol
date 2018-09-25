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

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type CurrentUser struct {
	UserName     string         `json:"userName" db:"username"`
	ID           int            `json:"id" db:"id"`
	TenantID     int            `json:"tenantId" db:"tenant_id"`
	Role         string         `json:"role" db:"role"`
	Capabilities pq.StringArray `json:"capabilities" db:"capabilities"`
}

// HasCapability returns whether this user has the given capability. Note capabilities are case-sensitive.
func (u *CurrentUser) HasCapability(c string) bool {
	for _, uc := range ([]string)(u.Capabilities) {
		if uc == c {
			return true
		}
	}
	return false
}

type PasswordForm struct {
	Username string `json:"u"`
	Password string `json:"p"`
}

const disallowed = "disallowed"

// TenantIDInvalid - The default Tenant ID
const TenantIDInvalid = -1

type key int

const CurrentUserKey key = iota

// GetCurrentUserFromDB returns the user info, a user-facing error, a system error to log, and an error code if there was an error.
func GetCurrentUserFromDB(db *sqlx.DB, userName string, timeout time.Duration) (CurrentUser, error, error, int) {
	qry := `
SELECT
  r.name as role,
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
	dbCtx, dbClose := context.WithTimeout(context.Background(), timeout)
	defer dbClose()

	user := CurrentUser{}
	if err := db.GetContext(dbCtx, &user, qry, userName); err != nil {
		if err == sql.ErrNoRows {
			return CurrentUser{}, errors.New("user not found"), errors.New("checking user " + userName + " info: user not in database"), http.StatusUnauthorized
		}
		if err == context.DeadlineExceeded || err == context.Canceled {
			return CurrentUser{}, nil, fmt.Errorf("db access timed out: %s number of open connections: %d\n", err, db.Stats().OpenConnections), http.StatusServiceUnavailable
		}
		return CurrentUser{}, nil, fmt.Errorf("checking user %v info: %v", userName, err.Error()), http.StatusInternalServerError
	}
	return user, nil, nil, http.StatusOK
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
	return &CurrentUser{}, errors.New("No user found in Context")
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
		if hashedPassword == sha1Hex(form.Password) { // for backwards compatibility
			return true, nil, nil
		}
		return false, err, nil
	}
	return true, nil, nil
}

func sha1Hex(s string) string {
	// SHA1 hash
	hash := sha1.New()
	hash.Write([]byte(s))
	hashBytes := hash.Sum(nil)

	// Hexadecimal conversion
	hexSha1 := hex.EncodeToString(hashBytes)
	return hexSha1
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
