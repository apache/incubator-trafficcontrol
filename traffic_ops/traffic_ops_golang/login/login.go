package login

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
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tocookie"

	"github.com/dgrijalva/jwt-go"
	"github.com/jmoiron/sqlx"
	"github.com/lestrrat-go/jwx/jwk"
)

type emailFormatter struct {
	From         rfc.EmailAddress
	To           rfc.EmailAddress
	InstanceName string
	ResetURL     string
	Token        string
}

const instanceNameQuery = `
SELECT value
FROM parameter
WHERE name='tm.instance_name' AND
      config_file=$1
`
const userQueryByEmail = `SELECT EXISTS(SELECT * FROM tm_user WHERE email=$1)`
const setTokenQuery = `UPDATE tm_user SET token=$1 WHERE email=$2`
const UpdateLoginTimeQuery = `UPDATE tm_user SET last_authenticated = now() WHERE username=$1`

const defaultCookieDuration = 6 * time.Hour

var resetPasswordEmailTemplate = template.Must(template.New("Password Reset Email").Parse("From: {{.From.Address.Address}}\r" + `
To: {{.To.Address.Address}}` + "\r" + `
Content-Type: text/html` + "\r" + `
Subject: {{.InstanceName}} Password Reset Request` + "\r\n\r" + `
<!DOCTYPE html>
<html lang="en">
<head>
	<title>{{.InstanceName}} Password Reset Request</title>
	<meta charset="utf-8"/>
	<style>
		.button_link {
			display: block;
			width: 130px;
			height: 35px;
			background: #2682AF;
			padding: 5px;
			text-align: center;
			border-radius: 5px;
			color: white;
			font-weight: bold;
			text-decoration: none;
			cursor: pointer;
		}
	</style>
</head>
<body>
  	<main>
  		<p>Someone has requested to change your password for the {{.InstanceName}}. If you requested this change, please click the link below and change your password. Otherwise, you can disregard this email.</p>
		<p><a class="button_link" target="_blank" href="{{.ResetURL}}?token={{.Token}}">Click to Reset Your Password</a></p>
	</main>
	<footer>
		<p>Thank you,<br/>
		The {{.InstanceName}} Team</p>
	</footer>
</body>
</html>
`))

func LoginHandler(db *sqlx.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErrs := tc.GetHandleErrorsFunc(w, r)
		defer r.Body.Close()
		authenticated := false
		form := auth.PasswordForm{}
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			handleErrs(http.StatusBadRequest, err)
			return
		}
		if form.Username == "" || form.Password == "" {
			api.HandleErr(w, r, nil, http.StatusBadRequest, errors.New("username and password are required"), nil)
			return
		}
		resp := struct {
			tc.Alerts
		}{}
		userAllowed, err, blockingErr := auth.CheckLocalUserIsAllowed(form, db, time.Duration(cfg.DBQueryTimeoutSeconds)*time.Second)
		if blockingErr != nil {
			api.HandleErr(w, r, nil, http.StatusServiceUnavailable, nil, fmt.Errorf("error checking local user password: %s\n", blockingErr.Error()))
			return
		}
		if err != nil {
			log.Errorf("checking local user: %s\n", err.Error())
		}
		if userAllowed {
			authenticated, err, blockingErr = auth.CheckLocalUserPassword(form, db, time.Duration(cfg.DBQueryTimeoutSeconds)*time.Second)
			if blockingErr != nil {
				api.HandleErr(w, r, nil, http.StatusServiceUnavailable, nil, fmt.Errorf("error checking local user password: %s\n", blockingErr.Error()))
				return
			}
			if err != nil {
				log.Errorf("checking local user password: %s\n", err.Error())
			}
			var ldapErr error
			if !authenticated {
				if cfg.LDAPEnabled {
					authenticated, ldapErr = auth.CheckLDAPUser(form, cfg.ConfigLDAP)
					if ldapErr != nil {
						log.Errorf("checking ldap user: %s\n", ldapErr.Error())
					}
				}
			}
			if authenticated {
				httpCookie := tocookie.GetCookie(form.Username, defaultCookieDuration, cfg.Secrets[0])
				http.SetCookie(w, httpCookie)

				//If all's well until here, then update last authenticated time
				tx, txErr := db.Begin()
				if txErr != nil {
					api.HandleErr(w, r, tx, http.StatusInternalServerError, nil, fmt.Errorf("beginning transaction: %w", txErr))
					return
				}
				defer tx.Commit()
				_, dbErr := tx.Exec(UpdateLoginTimeQuery, form.Username)
				if dbErr != nil {
					log.Errorf("unable to update authentication time for a given user: %s\n", dbErr.Error())
					resp = struct {
						tc.Alerts
					}{tc.CreateAlerts(tc.ErrorLevel, "Unable to update authentication time for a given user")}
				} else {
					resp = struct {
						tc.Alerts
					}{tc.CreateAlerts(tc.SuccessLevel, "Successfully logged in.")}
				}

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
		w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
		if !authenticated {
			w.WriteHeader(http.StatusUnauthorized)
		}
		fmt.Fprintf(w, "%s", respBts)
	}
}

func TokenLoginHandler(db *sqlx.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var t tc.UserToken
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			api.HandleErr(w, r, nil, http.StatusBadRequest, fmt.Errorf("Invalid request: %v", err), nil)
			return
		}

		tokenMatches, username, err := auth.CheckLocalUserToken(t.Token, db, time.Duration(cfg.DBQueryTimeoutSeconds)*time.Second)
		if err != nil {
			sysErr := fmt.Errorf("Checking token: %v", err)
			errCode := http.StatusInternalServerError
			api.HandleErr(w, r, nil, errCode, nil, sysErr)
			return
		} else if !tokenMatches {
			userErr := errors.New("Invalid token. Please contact your administrator.")
			errCode := http.StatusUnauthorized
			api.HandleErr(w, r, nil, errCode, userErr, nil)
			return
		}

		httpCookie := tocookie.GetCookie(username, defaultCookieDuration, cfg.Secrets[0])
		http.SetCookie(w, httpCookie)
		respBts, err := json.Marshal(tc.CreateAlerts(tc.SuccessLevel, "Successfully logged in."))
		if err != nil {
			sysErr := fmt.Errorf("Marshaling response: %v", err)
			errCode := http.StatusInternalServerError
			api.HandleErr(w, r, nil, errCode, nil, sysErr)
			return
		}

		w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
		api.WriteAndLogErr(w, r, append(respBts, '\n'))

		// TODO: afaik, Perl never clears these tokens. They should be reset to NULL on login, I think.
	}
}

// OauthLoginHandler accepts a JSON web token previously obtained from an OAuth provider, decodes it, validates it, authorizes the user against the database, and returns the login result as either an error or success message
func OauthLoginHandler(db *sqlx.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleErrs := tc.GetHandleErrorsFunc(w, r)
		defer r.Body.Close()
		authenticated := false
		resp := struct {
			tc.Alerts
		}{}

		form := auth.PasswordForm{}
		parameters := struct {
			AuthCodeTokenUrl string `json:"authCodeTokenUrl"`
			Code             string `json:"code"`
			ClientId         string `json:"clientId"`
			RedirectUri      string `json:"redirectUri"`
		}{}

		if err := json.NewDecoder(r.Body).Decode(&parameters); err != nil {
			handleErrs(http.StatusBadRequest, err)
			return
		}

		data := url.Values{}
		data.Add("code", parameters.Code)
		data.Add("client_id", parameters.ClientId)
		data.Add("grant_type", "authorization_code") // Required by RFC6749 section 4.1.3
		data.Add("redirect_uri", parameters.RedirectUri)

		req, err := http.NewRequest(http.MethodPost, parameters.AuthCodeTokenUrl, bytes.NewBufferString(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if cfg.OAuthClientSecret != "" {
			req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(parameters.ClientId+":"+cfg.OAuthClientSecret))) // per RFC6749 section 2.3.1
		}
		if err != nil {
			log.Errorf("obtaining token using code from oauth provider: %s", err.Error())
			return
		}

		client := http.Client{
			Timeout: 30 * time.Second,
		}
		response, err := client.Do(req)
		if err != nil {
			log.Errorf("getting an http client: %s", err.Error())
			return
		}
		defer response.Body.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(response.Body)
		encodedToken := ""

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			log.Warnf("Error parsing JSON response from oAuth: %s", err.Error())
			encodedToken = buf.String()
		} else if _, ok := result["access_token"]; !ok {
			sysErr := fmt.Errorf("Missing access token in response: %s\n", buf.String())
			usrErr := errors.New("Bad response from OAuth2.0 provider")
			api.HandleErr(w, r, nil, http.StatusBadGateway, usrErr, sysErr)
			return
		} else {
			switch t := result["access_token"].(type) {
			case string:
				encodedToken = result["access_token"].(string)
			default:
				sysErr := fmt.Errorf("Incorrect type of access_token! Expected 'string', got '%v'\n", t)
				usrErr := errors.New("Bad response from OAuth2.0 provider")
				api.HandleErr(w, r, nil, http.StatusBadGateway, usrErr, sysErr)
				return
			}
		}

		if encodedToken == "" {
			log.Errorf("Token not found in request but is required")
			handleErrs(http.StatusBadRequest, errors.New("Token not found in request but is required"))
			return
		}

		decodedToken, err := jwt.Parse(encodedToken, func(unverifiedToken *jwt.Token) (interface{}, error) {
			publicKeyUrl := unverifiedToken.Header["jku"].(string)
			publicKeyId := unverifiedToken.Header["kid"].(string)

			matched, err := VerifyUrlOnWhiteList(publicKeyUrl, cfg.ConfigTrafficOpsGolang.WhitelistedOAuthUrls)
			if err != nil {
				return nil, err
			}
			if !matched {
				return nil, errors.New("Key URL from token is not included in the whitelisted urls. Received: " + publicKeyUrl)
			}

			keys, err := jwk.FetchHTTP(publicKeyUrl)
			if err != nil {
				return nil, errors.New("Error fetching JSON key set with message: " + err.Error())
			}

			keyById := keys.LookupKeyID(publicKeyId)
			if len(keyById) == 0 {
				return nil, errors.New("No public key found for id: " + publicKeyId + " at url: " + publicKeyUrl)
			}

			selectedKey, err := keyById[0].Materialize()
			if err != nil {
				return nil, errors.New("Error materializing key from JSON key set with message: " + err.Error())
			}

			return selectedKey, nil
		})
		if err != nil {
			handleErrs(http.StatusInternalServerError, errors.New("Error decoding token with message: "+err.Error()))
			log.Errorf("Error decoding token: %s\n", err.Error())
			return
		}

		authenticated = decodedToken.Valid

		userId := decodedToken.Claims.(jwt.MapClaims)["sub"].(string)
		form.Username = userId

		userAllowed, err, blockingErr := auth.CheckLocalUserIsAllowed(form, db, time.Duration(cfg.DBQueryTimeoutSeconds)*time.Second)
		if blockingErr != nil {
			api.HandleErr(w, r, nil, http.StatusServiceUnavailable, nil, fmt.Errorf("error checking local user password: %s\n", blockingErr.Error()))
			return
		}
		if err != nil {
			log.Errorf("checking local user: %s\n", err.Error())
		}

		if userAllowed && authenticated {
			httpCookie := tocookie.GetCookie(userId, defaultCookieDuration, cfg.Secrets[0])
			http.SetCookie(w, httpCookie)
			resp = struct {
				tc.Alerts
			}{tc.CreateAlerts(tc.SuccessLevel, "Successfully logged in.")}
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
		w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
		if !authenticated {
			w.WriteHeader(http.StatusUnauthorized)
		}
		if !userAllowed {
			w.WriteHeader(http.StatusForbidden)
		}
		fmt.Fprintf(w, "%s", respBts)

	}
}

func VerifyUrlOnWhiteList(urlString string, whiteListedUrls []string) (bool, error) {

	for _, listing := range whiteListedUrls {
		if listing == "" {
			continue
		}

		urlParsed, err := url.Parse(urlString)
		if err != nil {
			return false, err
		}

		matched, err := filepath.Match(listing, urlParsed.Hostname())
		if err != nil {
			return false, err
		}

		if matched {
			return true, nil
		}
	}
	return false, nil
}

func generateToken() (string, error) {
	var t = make([]byte, 16)
	_, err := rand.Read(t)
	if err != nil {
		return "", err
	}
	t[6] = (t[6] & 0x0f) | 0x40
	t[8] = (t[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", t[0:4], t[4:6], t[6:8], t[8:10], t[10:]), nil
}

func setToken(addr rfc.EmailAddress, tx *sql.Tx) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	if _, err = tx.Exec(setTokenQuery, token, addr.Address.Address); err != nil {
		return "", err
	}
	return token, nil
}

func createMsg(addr rfc.EmailAddress, t string, db *sqlx.DB, c config.ConfigPortal) ([]byte, error) {
	var instanceName string
	row := db.QueryRow(instanceNameQuery, tc.GlobalConfigFileName)
	if err := row.Scan(&instanceName); err != nil {
		return nil, err
	}
	f := emailFormatter{
		From:         c.EmailFrom,
		To:           addr,
		Token:        t,
		InstanceName: instanceName,
		ResetURL:     c.BaseURL.String() + c.PasswdResetPath,
	}

	var tmpl bytes.Buffer
	if err := resetPasswordEmailTemplate.Execute(&tmpl, &f); err != nil {
		return nil, err
	}
	return tmpl.Bytes(), nil
}

func ResetPassword(db *sqlx.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userErr, sysErr error
		var errCode int
		tx, err := db.Begin()
		if err != nil {
			sysErr = fmt.Errorf("Beginning transaction: %v", err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		}
		defer r.Body.Close()

		var req tc.UserPasswordResetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			userErr = fmt.Errorf("Malformed request: %v", err)
			errCode = http.StatusBadRequest
			api.HandleErr(w, r, tx, errCode, userErr, nil)
			return
		}

		row := tx.QueryRow(userQueryByEmail, req.Email.Address.Address)
		var userExists bool
		if err := row.Scan(&userExists); err != nil {
			sysErr = fmt.Errorf("Checking for existence of user with email '%s': %v", req.Email.String(), err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		}
		if !userExists {
			// TODO: consider concealing database state from unauthenticated parties;
			// this should maybe just return a 2XX w/ success message at this point?
			userErr = fmt.Errorf("No account with the email address '%s' was found!", req.Email.Address.Address)
			errCode = http.StatusNotFound
			api.HandleErr(w, r, tx, errCode, userErr, nil)
			return
		}

		token, err := setToken(req.Email, tx)
		if err != nil {
			sysErr = fmt.Errorf("Failed to generate and insert UUID: %v", err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, tx, errCode, nil, sysErr)
			return
		}
		tx.Commit()

		msg, err := createMsg(req.Email, token, db, cfg.ConfigPortal)
		if err != nil {
			sysErr = fmt.Errorf("Failed to create email message: %v", err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, nil, errCode, nil, sysErr)
			return
		}

		log.Debugf("Sending password reset email to %s", req.Email)

		if errCode, userErr, sysErr = api.SendMail(req.Email, msg, &cfg); userErr != nil || sysErr != nil {
			api.HandleErr(w, r, nil, errCode, userErr, sysErr)
			return
		}

		alerts := tc.CreateAlerts(tc.SuccessLevel, "Password reset email sent")
		respBts, err := json.Marshal(alerts)
		if err != nil {
			userErr = errors.New("Email was sent, but an error occurred afterward")
			sysErr = fmt.Errorf("Marshaling response: %v", err)
			errCode = http.StatusInternalServerError
			api.HandleErr(w, r, nil, errCode, userErr, sysErr)
			return
		}

		w.Header().Set(rfc.ContentType, rfc.ApplicationJSON)
		api.WriteAndLogErr(w, r, append(respBts, '\n'))
	}
}
