package api

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
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/trafficvault/backends/disabled"

	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type tester struct {
	ID         int
	InfoerImpl `json:"-"`
	userErr    error //only for testing
	sysErr     error //only for testing
	errCode    int   //only for testing
}

var cfg = config.Config{ConfigTrafficOpsGolang: config.ConfigTrafficOpsGolang{DBQueryTimeoutSeconds: 20}, UseIMS: true}

func (i tester) GetKeyFieldsInfo() []KeyFieldInfo {
	return []KeyFieldInfo{{"id", GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (i tester) GetKeys() (map[string]interface{}, bool) {
	return map[string]interface{}{"id": i.ID}, true
}

func (i *tester) SetKeys(keys map[string]interface{}) {
	id, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	i.ID = id
}

func (i *tester) GetType() string {
	return "tester"
}

func (i *tester) GetAuditName() string {
	return "testerInstance:" + strconv.Itoa(i.ID)
}

//Validator interface function
func (v *tester) Validate() error {
	if v.ID < 1 {
		return errors.New("ID is too low")
	}
	return nil
}

//Creator interface functions
func (i *tester) Create() (error, error, int) {
	return i.userErr, i.sysErr, i.errCode
}

//Reader interface functions
func (i *tester) Read(h http.Header, useIMS bool) ([]interface{}, error, error, int, *time.Time) {
	if h.Get(rfc.IfModifiedSince) != "" {
		if imsDate, ok := rfc.ParseHTTPDate(h.Get(rfc.IfModifiedSince)); !ok {
			return []interface{}{tester{ID: 1}}, nil, nil, http.StatusOK, nil
		} else {
			if imsDate.UTC().After(time.Now().UTC()) {
				return []interface{}{}, nil, nil, http.StatusNotModified, &imsDate
			}
		}
	}
	return []interface{}{tester{ID: 1}}, nil, nil, http.StatusOK, nil
}

//Updater interface functions
func (i *tester) Update(http.Header) (error, error, int) {
	return i.userErr, i.sysErr, i.errCode
}

//Deleter interface functions
func (i *tester) Delete() (error, error, int) {
	return i.userErr, i.sysErr, i.errCode
}

//used for testing purposes only
func (t *tester) SetError(userErr error, sysErr error, errCode int) {
	t.userErr = userErr
	t.sysErr = sysErr
	t.errCode = errCode
}

func TestCreateHandler(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", "", strings.NewReader(`{"ID":1}`))
	if err != nil {
		t.Error("Error creating new request")
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, auth.CurrentUserKey,
		auth.CurrentUser{UserName: "username", ID: 1, PrivLevel: auth.PrivLevelAdmin})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	var tv trafficvault.TrafficVault = &disabled.Disabled{}
	ctx = context.WithValue(ctx, TrafficVaultContextKey, tv)

	// Add our context to the request
	r = r.WithContext(ctx)

	typeRef := &tester{ID: 1}
	createFunc := CreateHandler(typeRef)

	//verifies we get the right changelog insertion
	keys, _ := typeRef.GetKeys()
	expectedMessage := strings.ToUpper(typeRef.GetType()) + ": " + typeRef.GetAuditName() + ", ID: " + strconv.Itoa(keys["id"].(int)) + ", ACTION: " + Created + " " + typeRef.GetType() + ", keys: { id:" + strconv.Itoa(keys["id"].(int)) + " }"
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WithArgs(ApiChange, expectedMessage, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	createFunc(w, r)

	//verifies the body is in the expected format
	body := `{"alerts":[{"text":"tester was created.","level":"success"}],"response":{"ID":1}}` + "\n"
	if w.Body.String() != body {
		t.Error("Expected body", body, "got", w.Body.String())
	}
}

func TestReadHandler(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", "", nil)
	if err != nil {
		t.Error("Error creating new request")
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, auth.CurrentUserKey,
		auth.CurrentUser{UserName: "username", ID: 1, PrivLevel: auth.PrivLevelAdmin})
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))
	var tv trafficvault.TrafficVault = &disabled.Disabled{}
	ctx = context.WithValue(ctx, TrafficVaultContextKey, tv)

	// Add our context to the request
	r = r.WithContext(ctx)
	readFunc := ReadHandler(&tester{})

	mock.ExpectBegin()
	mock.ExpectCommit()

	readFunc(w, r)

	//verifies the body is in the expected format
	body := `{"response":[{"ID":1}]}` + "\n"
	if w.Body.String() != body {
		t.Error("Expected body", body, "got", w.Body.String())
	}
	if w.Result().Header.Get(rfc.LastModified) != "" {
		t.Errorf("Expected no last modified header (since this is a non IMS request), but got %v", w.Result().Header.Get(rfc.LastModified))
	}
}

func TestReadHandlerIMS(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", "", nil)
	if err != nil {
		t.Error("Error creating new request")
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, auth.CurrentUserKey,
		auth.CurrentUser{UserName: "username", ID: 1, PrivLevel: auth.PrivLevelAdmin})
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))
	var tv trafficvault.TrafficVault = &disabled.Disabled{}
	ctx = context.WithValue(ctx, TrafficVaultContextKey, tv)
	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)
	r.Header.Add(rfc.IfModifiedSince, time)
	// Add our context to the request
	r = r.WithContext(ctx)
	readFunc := ReadHandler(&tester{})

	mock.ExpectBegin()
	mock.ExpectCommit()

	readFunc(w, r)

	if w.Result().StatusCode != http.StatusNotModified {
		t.Errorf("Expected status code of 304, got %v instead", w.Result().StatusCode)
	}
	if w.Result().Header.Get(rfc.LastModified) == "" {
		t.Error("Expected a valid last modified header, but got nothing")
	}
}

func TestUpdateHandler(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", "", strings.NewReader(`{"ID":1}`))
	if err != nil {
		t.Error("Error creating new request")
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, auth.CurrentUserKey,
		auth.CurrentUser{UserName: "username", ID: 1, PrivLevel: auth.PrivLevelAdmin})
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))
	var tv trafficvault.TrafficVault = &disabled.Disabled{}
	ctx = context.WithValue(ctx, TrafficVaultContextKey, tv)

	// Add our context to the request
	r = r.WithContext(ctx)

	typeRef := &tester{ID: 1}
	updateFunc := UpdateHandler(typeRef)

	//verifies we get the right changelog insertion
	keys, _ := typeRef.GetKeys()
	expectedMessage := strings.ToUpper(typeRef.GetType()) + ": " + typeRef.GetAuditName() + ", ID: " + strconv.Itoa(keys["id"].(int)) + ", ACTION: " + Updated + " " + typeRef.GetType() + ", keys: { id:" + strconv.Itoa(keys["id"].(int)) + " }"
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WithArgs(ApiChange, expectedMessage, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	updateFunc(w, r)

	//verifies the body is in the expected format
	body := `{"alerts":[{"text":"tester was updated.","level":"success"}],"response":{"ID":1}}` + "\n"
	if w.Body.String() != body {
		t.Error("Expected body", body, "got", w.Body.String())
	}
}

func TestDeleteHandler(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	w := httptest.NewRecorder()
	r, err := http.NewRequest("", "", strings.NewReader(`{"ID":1}`))
	if err != nil {
		t.Error("Error creating new request")
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, auth.CurrentUserKey,
		auth.CurrentUser{UserName: "username", ID: 1, PrivLevel: auth.PrivLevelAdmin})
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))
	var tv trafficvault.TrafficVault = &disabled.Disabled{}
	ctx = context.WithValue(ctx, TrafficVaultContextKey, tv)
	// Add our context to the request
	r = r.WithContext(ctx)

	typeRef := &tester{ID: 1}
	deleteFunc := DeleteHandler(typeRef)

	//verifies we get the right changelog insertion
	keys, _ := typeRef.GetKeys()
	expectedMessage := strings.ToUpper(typeRef.GetType()) + ": " + typeRef.GetAuditName() + ", ID: " + strconv.Itoa(keys["id"].(int)) + ", ACTION: " + Deleted + " " + typeRef.GetType() + ", keys: { id:" + strconv.Itoa(keys["id"].(int)) + " }"
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WithArgs(ApiChange, expectedMessage, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	deleteFunc(w, r)

	//verifies the body is in the expected format
	body := `{"alerts":[{"text":"tester was deleted.","level":"success"}]}` + "\n"
	if w.Body.String() != body {
		t.Error("Expected body", body, "got", w.Body.String())
	}
}
