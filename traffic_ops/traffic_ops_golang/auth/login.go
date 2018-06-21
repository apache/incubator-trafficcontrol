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
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tocookie"

	"github.com/jmoiron/sqlx"
)

const disallowed = "disallowed"

type passwordForm struct {
	Username string `json:"u"`
	Password string `json:"p"`
}

func LoginHandler(db *sqlx.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErrs := tc.GetHandleErrorsFunc(w, r)
		defer r.Body.Close()
		authenticated := false
		form := passwordForm{}
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			handleErrs(http.StatusBadRequest, err)
			return
		}
		resp := struct {
			tc.Alerts
		}{}
		userAllowed, err := CheckLocalUserIsAllowed(form, db)
		if err != nil {
			log.Errorf("error checking local user: %s\n", err.Error())
		}
		if userAllowed {
			authenticated, err = checkLocalUserPassword(form, db)
			if err != nil {
				log.Errorf("error checking local user password: %s\n", err.Error())
			}
			var ldapErr error
			if !authenticated {
				if cfg.LDAPEnabled {
					authenticated, ldapErr = checkLDAPUser(form, cfg.ConfigLDAP)
					if ldapErr != nil {
						log.Errorf("error checking ldap user: %s\n", ldapErr.Error())
					}
				}
			}
			if authenticated {
				expiry := time.Now().Add(time.Hour * 6)
				cookie := tocookie.New(form.Username, expiry, cfg.Secrets[0])
				httpCookie := http.Cookie{Name: "mojolicious", Value: cookie, Path: "/", Expires: expiry, HttpOnly: true}
				http.SetCookie(w, &httpCookie)
				resp = struct {
					tc.Alerts
				}{tc.CreateAlerts(tc.SuccessLevel, "Successfully logged in.")}
			} else {
				resp = struct {
					tc.Alerts
				}{tc.CreateAlerts(tc.ErrorLevel, "Invalid username or password.")}
			}
		} else {
			resp = struct {
				tc.Alerts
			}{tc.CreateAlerts(tc.ErrorLevel, "Invalid username or password.")}
		}
		respBts, err := json.Marshal(resp)
		if err != nil {
			handleErrs(http.StatusInternalServerError, err)
			return
		}
		w.Header().Set(tc.ContentType, tc.ApplicationJson)
		if !authenticated {
			w.WriteHeader(http.StatusUnauthorized)
		}
		fmt.Fprintf(w, "%s", respBts)
	}
}

func CheckLocalUserIsAllowed(form passwordForm, db *sqlx.DB) (bool, error) {
	var roleName string
	err := db.Get(&roleName, "SELECT role.name FROM role INNER JOIN tm_user ON tm_user.role = role.id where username=$1", form.Username)
	if err != nil {
		return false, err
	}
	if roleName != "" {
		if roleName != disallowed { //relies on unchanging role name assumption.
			return true, nil
		}
	}
	return false, nil
}

func checkLocalUserPassword(form passwordForm, db *sqlx.DB) (bool, error) {
	var hashedPassword string
	err := db.Get(&hashedPassword, "SELECT local_passwd FROM tm_user WHERE username=$1", form.Username)
	if err != nil {
		return false, err
	}
	err = VerifySCRYPTPassword(form.Password, hashedPassword)
	if err != nil {
		if hashedPassword == sha1Hex(form.Password) { // for backwards compatibility
			return true, nil
		}
		return false, err
	}
	return true, nil
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

func checkLDAPUser(form passwordForm, cfg *config.ConfigLDAP) (bool, error) {
	userDN, valid, err := LookupUserDN(form.Username, cfg)
	if err != nil {
		return false, err
	}
	if valid {
		return AuthenticateUserDN(userDN, form.Password, cfg)
	}
	return false, errors.New("User not found in LDAP")
}
