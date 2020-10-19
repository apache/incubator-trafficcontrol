package federations

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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestAddFederationResolverMappingsForCurrentUser(t *testing.T) {
	t.Run("add Federation Resolver Mappings for the current user", positiveTestAddFederationResolverMappingsForCurrentUser)
	t.Run("add Federation Resolver Mappings for the current user when no federations exist/are assigned to them", testAddFederationResolverMappingsForCurrentUserWithoutFederations)
	t.Run("add Federation Resolver Mappings for a DS unauthorized to the current user's tenant", testUnauthorizedDSOnResolverAdd)
}

func positiveTestAddFederationResolverMappingsForCurrentUser(t *testing.T) {
	u := auth.CurrentUser{
		UserName:  "test",
		ID:        1,
		PrivLevel: 100,
		TenantID:  1,
		Role:      1,
	}

	mappings := []tc.DeliveryServiceFederationResolverMapping{
		tc.DeliveryServiceFederationResolverMapping{
			DeliveryService: "test",
			Mappings: tc.ResolverMapping{
				Resolve4: []string{"0.0.0.0", "127.0.0.1/12"},
				Resolve6: []string{"abcd:ef01:2345:6789::", "f1d0::f00d/127"},
			},
		},
	}

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	dsTenantIDRows := sqlmock.NewRows([]string{"tenant_id"})
	dsTenantIDRows.AddRow(1)
	authorizedRows := sqlmock.NewRows([]string{"id", "active"})
	authorizedRows.AddRow(1, true)
	fedIDRows := sqlmock.NewRows([]string{"federation"})
	fedIDRows.AddRow(1)
	insertFirstResolverRows := sqlmock.NewRows([]string{"ip_address", "id"})
	insertFirstResolverRows.AddRow(mappings[0].Mappings.Resolve4[0], 1)
	insertSecondResolverRows := sqlmock.NewRows([]string{"ip_address", "id"})
	insertSecondResolverRows.AddRow(mappings[0].Mappings.Resolve4[1], 2)
	insertThirdResolverRows := sqlmock.NewRows([]string{"ip_address", "id"})
	insertThirdResolverRows.AddRow(mappings[0].Mappings.Resolve6[0], 3)
	insertFourthResolverRows := sqlmock.NewRows([]string{"ip_address", "id"})
	insertFourthResolverRows.AddRow(mappings[0].Mappings.Resolve6[1], 4)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT deliveryservice.tenant_id").WillReturnRows(dsTenantIDRows)
	mock.ExpectQuery("WITH RECURSIVE").WillReturnRows(authorizedRows)
	mock.ExpectQuery("SELECT federation_deliveryservice.federation").WillReturnRows(fedIDRows)
	mock.ExpectQuery("INSERT INTO federation_resolver").WillReturnRows(insertFirstResolverRows)
	mock.ExpectExec("INSERT INTO federation_federation_resolver").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("INSERT INTO federation_resolver").WillReturnRows(insertSecondResolverRows)
	mock.ExpectExec("INSERT INTO federation_federation_resolver").WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectQuery("INSERT INTO federation_resolver").WillReturnRows(insertThirdResolverRows)
	mock.ExpectExec("INSERT INTO federation_federation_resolver").WillReturnResult(sqlmock.NewResult(3, 1))
	mock.ExpectQuery("INSERT INTO federation_resolver").WillReturnRows(insertFourthResolverRows)
	mock.ExpectExec("INSERT INTO federation_federation_resolver").WillReturnResult(sqlmock.NewResult(4, 1))
	mock.ExpectCommit()

	tx, err := mockDB.Begin()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when beginning a mock transaction", err)
	}

	errs := addFederationResolverMappingsForCurrentUser(&u, tx, mappings)
	if errs.Occurred() {
		t.Errorf("Unexpected error(s): %s", errs)
	}
	if errs.Code != http.StatusOK {
		t.Errorf("Expected response code %d, got %d", http.StatusOK, errs.Code)
	}
}

func testAddFederationResolverMappingsForCurrentUserWithoutFederations(t *testing.T) {
	u := auth.CurrentUser{
		UserName:  "test",
		ID:        1,
		PrivLevel: 100,
		TenantID:  1,
		Role:      1,
	}

	mappings := []tc.DeliveryServiceFederationResolverMapping{
		tc.DeliveryServiceFederationResolverMapping{
			DeliveryService: "test",
			Mappings: tc.ResolverMapping{
				Resolve4: []string{"0.0.0.0"},
				Resolve6: []string{},
			},
		},
	}

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	dsTenantIDRows := sqlmock.NewRows([]string{"tenant_id"})
	dsTenantIDRows.AddRow(1)
	authorizedRows := sqlmock.NewRows([]string{"id", "active"})
	authorizedRows.AddRow(1, true)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT deliveryservice.tenant_id").WillReturnRows(dsTenantIDRows)
	mock.ExpectQuery("WITH RECURSIVE").WillReturnRows(authorizedRows)
	mock.ExpectQuery("SELECT federation_deliveryservice.federation").WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()

	tx, err := mockDB.Begin()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when beginning a mock transaction", err)
	}

	errs := addFederationResolverMappingsForCurrentUser(&u, tx, mappings)
	if errs.UserError == nil {
		t.Errorf("Unexpected a user error, but didn't get one")
	} else {
		t.Logf("Got expected user error: %v", errs.UserError)
	}
	if errs.SystemError != nil {
		t.Errorf("Unexpected system error: %v", errs.SystemError)
	}
	if errs.Code != http.StatusConflict {
		t.Errorf("Expected response code %d, got %d", http.StatusConflict, errs.Code)
	}
}

func testUnauthorizedDSOnResolverAdd(t *testing.T) {
	u := auth.CurrentUser{
		UserName:  "test",
		ID:        1,
		PrivLevel: 100,
		TenantID:  1,
		Role:      1,
	}

	mappings := []tc.DeliveryServiceFederationResolverMapping{
		tc.DeliveryServiceFederationResolverMapping{
			DeliveryService: "test",
			Mappings: tc.ResolverMapping{
				Resolve4: []string{},
				Resolve6: []string{},
			},
		},
	}

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	dsTenantIDRows := sqlmock.NewRows([]string{"tenant_id"})
	dsTenantIDRows.AddRow(1)
	authorizedRows := sqlmock.NewRows([]string{"id", "active"})
	authorizedRows.AddRow(1, false) // unauthorized because tenant is inactive

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT deliveryservice.tenant_id").WillReturnRows(dsTenantIDRows)
	mock.ExpectQuery("WITH RECURSIVE").WillReturnRows(authorizedRows)
	mock.ExpectCommit()

	tx, err := mockDB.Begin()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when beginning a mock transaction", err)
	}

	errs := addFederationResolverMappingsForCurrentUser(&u, tx, mappings)
	if errs.UserError == nil {
		t.Errorf("Unexpected a user error, but didn't get one")
	} else {
		t.Logf("Got expected user error: %v", errs.UserError)
	}
	if errs.SystemError == nil {
		t.Errorf("Unexpected a system error, but didn't get one")
	} else {
		t.Logf("Got expected system error: %v", errs.SystemError)
	}
	if errs.Code != http.StatusConflict {
		t.Errorf("Expected response code %d, got %d", http.StatusConflict, errs.Code)
	}
}

func TestGetMappingsFromRequestBody(t *testing.T) {
	data := `{"federations":[{"deliveryService":"test","mappings":{"resolve4":[], "resolve6":[]}}]}`
	buf := strings.NewReader(data)

	legacyVersion := api.Version{
		Major: 1,
		Minor: 3,
	}

	version := api.Version{
		Major: 1,
		Minor: 4,
	}

	_, userErr, sysErr := getMappingsFromRequestBody(legacyVersion, ioutil.NopCloser(buf))
	if userErr != nil {
		t.Errorf("Unexpected user error legacy parsing '%s': %v", data, userErr)
	}
	if sysErr != nil {
		t.Errorf("Unexpected system error legacy parsing '%s': %v", data, sysErr)
	}

	buf = strings.NewReader(data)

	_, userErr, sysErr = getMappingsFromRequestBody(version, ioutil.NopCloser(buf))
	if userErr != nil {
		t.Errorf("Unexpected user error parsing '%s': %v", data, userErr)
	}
	if sysErr != nil {
		t.Errorf("Unexpected system error parsing '%s': %v", data, sysErr)
	}

	data = `[{"deliveryService":"test","mappings":{"resolve4":[], "resolve6":[]}}]`
	buf = strings.NewReader(data)

	_, userErr, sysErr = getMappingsFromRequestBody(legacyVersion, ioutil.NopCloser(buf))
	if userErr == nil {
		t.Errorf("Expected user error legacy parsing '%s' but didn't get one", data)
	} else {
		t.Logf("Got expected user error legacy parsing '%s': %v", data, userErr)
	}
	if sysErr != nil {
		t.Errorf("Unexpected system error legacy parsing '%s': %v", data, sysErr)
	}

	buf = strings.NewReader(data)

	_, userErr, sysErr = getMappingsFromRequestBody(version, ioutil.NopCloser(buf))
	if userErr != nil {
		t.Errorf("Unexpected user error parsing '%s': %v", data, userErr)
	}
	if sysErr != nil {
		t.Errorf("Unexpected system error parsing '%s': %v", data, sysErr)
	}
}

func TestRemoveFederationResolverMappingsForCurrentUser(t *testing.T) {
	u := auth.CurrentUser{
		UserName:  "test",
		ID:        1,
		PrivLevel: 100,
		TenantID:  1,
		Role:      1,
	}

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	ips := []string{
		"0.0.0.0",
		"127.0.0.1/32",
		"::1",
		"::1/64",
	}

	rows := sqlmock.NewRows([]string{"ip_address"})
	rows.AddRow(ips[0])
	rows.AddRow(ips[1])
	rows.AddRow(ips[2])
	rows.AddRow(ips[3])

	mock.ExpectBegin()
	mock.ExpectQuery("DELETE").WillReturnRows(rows)
	mock.ExpectCommit()

	tx, err := mockDB.Begin()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when beginning a mock transaction", err)
	}

	returnedIPs, errs := removeFederationResolverMappingsForCurrentUser(tx, &u)
	if errs.Occurred() {
		t.Errorf("Unexpected error(s) removing resolvers: %s", errs)
	}
	if errs.Code != http.StatusOK {
		t.Errorf("Expected return code %d when removing resolvers, but got %d", http.StatusOK, errs.Code)
	}

	if len(returnedIPs) != len(ips) {
		t.Fatalf("Length of returned IP array (%d) does not match expected length (%d)", len(returnedIPs), len(ips))
	}

	for i, ip := range ips {
		if returnedIPs[i] != ip {
			t.Errorf("Returned IP #%d was '%s', but '%s' was expected", i, returnedIPs[i], ip)
		}
	}
}
