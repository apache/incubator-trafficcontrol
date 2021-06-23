// Package acme contains logic and handlers pertaining to the /acme_accounts,
// /acme_accounts/providers, and /acme_accounts/{{provider}}/{{email}} API
// endpoints.
package acme

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
	"net/url"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
)

const readQuery = `SELECT email, private_key, uri, provider FROM acme_account`
const readProvidersQuery = `SELECT DISTINCT provider FROM acme_account`
const createQuery = `INSERT INTO acme_account (email, private_key, uri, provider) VALUES (:email, :private_key, :uri, :provider) RETURNING email, provider`
const updateQuery = `UPDATE acme_account SET private_key=:private_key, uri=:uri WHERE email=:email and provider=:provider RETURNING email, provider`
const deleteQuery = `DELETE FROM acme_account WHERE email=$1 and provider=$2`
const selectByProviderAndEmailQuery = `SELECT email, private_key, uri, provider from acme_account where email = $1 and provider = $2`
const selectLimitedQuery = `SELECT email, provider from acme_account where email = $1 and provider = $2`

// Read is the handler for GET requests to /acme_accounts.
func Read(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	params := map[string]dbhelpers.WhereColumnInfo{
		"email":    {Column: "email"},
		"provider": {Column: "provider"},
	}

	rows, errCode, userErr, sysErr := inf.GetFilteredRows(readQuery, params)
	if userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("querying ACME acounts: %v", err)
		}
	}()

	acmeAccounts := []tc.AcmeAccount{}
	for rows.Next() {
		var acct tc.AcmeAccount
		if err := rows.StructScan(&acct); err != nil {
			inf.HandleErr(http.StatusInternalServerError, nil, fmt.Errorf("scanning acme accounts: %w", err))
			return
		}
		acmeAccounts = append(acmeAccounts, acct)
	}

	inf.WriteOKResponse(acmeAccounts, nil)
}

// ReadProviders returns a list of unique ACME provider both from the database and cdn.conf.
func ReadProviders(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	acmeProviders := []string{}
	rows, errCode, userErr, sysErr := inf.GetFilteredRows(readProvidersQuery, nil)
	if userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("querying ACME providers: %v", err)
		}
	}()

	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			inf.HandleErr(http.StatusInternalServerError, nil, errors.New("scanning acme account providers: "+err.Error()))
			return
		}
		acmeProviders = append(acmeProviders, provider)
	}

	for _, acmeCfg := range inf.Config.AcmeAccounts {
		alreadyInList := false
		for _, acmeProvider := range acmeProviders {
			if acmeCfg.AcmeProvider == acmeProvider {
				alreadyInList = true
			}
		}
		if !alreadyInList {
			acmeProviders = append(acmeProviders, acmeCfg.AcmeProvider)
		}
	}

	inf.WriteOKResponse(acmeProviders, nil)
}

// Create handles POST requests to add a new ACME provider.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	tx := inf.Tx.Tx

	var acmeAccount tc.AcmeAccount
	if userErr = inf.ParseAndValidateBody(&acmeAccount); userErr != nil {
		inf.HandleErr(http.StatusBadRequest, userErr, nil)
		return
	}

	var prevEmail string
	var prevProvider string
	err := tx.QueryRow(selectLimitedQuery, acmeAccount.Email, acmeAccount.Provider).Scan(&prevEmail, &prevProvider)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		inf.HandleErr(http.StatusInternalServerError, nil, fmt.Errorf("checking if acme account with email %s and provider %s exists: %w", *acmeAccount.Email, *acmeAccount.Provider, err))
		return
	}

	if prevEmail != "" && prevProvider != "" {
		inf.HandleErr(http.StatusBadRequest, errors.New("acme account already exists"), nil)
		return
	}

	if errCode, userErr, sysErr = inf.CreateOrUpdate(createQuery, acmeAccount); userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}

	inf.SetHeader(rfc.Location, fmt.Sprintf("/api/%s/acme_accounts?email=%s&provider=%s", inf.Version, url.QueryEscape(prevEmail), url.QueryEscape(prevProvider)))
	inf.WriteResponseWithAlert(acmeAccount, http.StatusCreated, tc.SuccessLevel, "Acme account created")
	inf.CreateChangeLog(fmt.Sprintf("ACME ACCOUNT: %s %s, ACTION: created", *acmeAccount.Email, *acmeAccount.Provider))
}

// Update is the handler for PUT requests to /acme_accounts.
func Update(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	tx := inf.Tx.Tx

	var acmeAccount tc.AcmeAccount
	if userErr = inf.ParseAndValidateBody(&acmeAccount); userErr != nil {
		inf.HandleErr(http.StatusBadRequest, userErr, nil)
		return
	}

	var prevAccount tc.AcmeAccount
	err := tx.QueryRow(selectByProviderAndEmailQuery, acmeAccount.Email, acmeAccount.Provider).Scan(&prevAccount.Email, &prevAccount.PrivateKey, &prevAccount.Uri, &prevAccount.Provider)
	if errors.Is(err, sql.ErrNoRows) {
		inf.HandleErr(http.StatusBadRequest, errors.New("acme account not found"), nil)
		return
	}
	if err != nil {
		inf.HandleErr(http.StatusInternalServerError, nil, fmt.Errorf("checking if acme account with email %s and provider %s exists: %w", *acmeAccount.Email, *acmeAccount.Provider, err))
		return
	}

	if errCode, userErr, sysErr = inf.CreateOrUpdate(updateQuery, acmeAccount); userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}

	inf.WriteResponseWithAlert(acmeAccount, http.StatusOK, tc.SuccessLevel, "Acme account updated")
	inf.CreateChangeLog(fmt.Sprintf("ACME ACCOUNT: %s %s, ACTION: updated", *acmeAccount.Email, *acmeAccount.Provider))
}

// Delete removes the information about an ACME account.
func Delete(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, []string{"provider", "email"}, nil)
	if userErr != nil || sysErr != nil {
		inf.HandleErr(errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	provider := inf.Params["provider"]
	email := inf.Params["email"]

	tx := inf.Tx.Tx

	var prevAccount tc.AcmeAccount
	err := tx.QueryRow(selectByProviderAndEmailQuery, email, provider).Scan(&prevAccount.Email, &prevAccount.PrivateKey, &prevAccount.Uri, &prevAccount.Provider)
	if errors.Is(err, sql.ErrNoRows) {
		inf.HandleErr(http.StatusBadRequest, errors.New("acme account not found"), nil)
		return
	}
	if err != nil {
		inf.HandleErr(http.StatusInternalServerError, nil, fmt.Errorf("checking if acme account with email %s and provider %s exists: %w", email, provider, err))
		return
	}

	if _, err := tx.Exec(deleteQuery, email, provider); err != nil {
		inf.HandleErr(http.StatusInternalServerError, nil, fmt.Errorf("deleting acme account with email %s and provider %s: %w", email, provider, err))
		return
	}

	inf.WriteResponseWithAlert(prevAccount, http.StatusOK, tc.SuccessLevel, "Acme account deleted")
	inf.CreateChangeLog(fmt.Sprintf("ACME ACCOUNT: %s %s, ACTION: deleted", email, provider))
}
