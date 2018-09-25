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

	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/jmoiron/sqlx"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type tester struct {
	ID      int
	userErr error //only for testing
	sysErr  error //only for testing
	errCode int   //only for testing
}

func GetTypeSingleton() func(apiInfo *APIInfo) CRUDer {
	return func(apiInfo *APIInfo) CRUDer {
		return &tester{}
	}
}

var cfg = config.Config{ConfigTrafficOpsGolang: config.ConfigTrafficOpsGolang{DBQueryTimeoutSeconds: 20}}

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
func (i *tester) Read() ([]interface{}, error, error, int) {
	return []interface{}{tester{ID: 1}}, nil, nil, http.StatusOK
}

//Updater interface functions
func (i *tester) Update() (error, error, int) {
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
		auth.CurrentUser{UserName: "username", ID: 1})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})

	// Add our context to the request
	r = r.WithContext(ctx)

	typeRef := tester{ID: 1}

	createFunc := CreateHandler(GetTypeSingleton())

	//verifies we get the right changelog insertion
	expectedMessage := Created + " " + typeRef.GetType() + ": " + typeRef.GetAuditName() + " keys: { id:1 }"
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WithArgs(ApiChange, expectedMessage, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	createFunc(w, r)

	//verifies the body is in the expected format
	body := `{"alerts":[{"text":"tester was created.","level":"success"}],"response":{"ID":1}}`
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
		auth.CurrentUser{UserName: "username", ID: 1})
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))

	// Add our context to the request
	r = r.WithContext(ctx)

	readFunc := ReadHandler(GetTypeSingleton())

	mock.ExpectBegin()
	mock.ExpectCommit()

	readFunc(w, r)

	//verifies the body is in the expected format
	body := `{"response":[{"ID":1}]}`
	if w.Body.String() != body {
		t.Error("Expected body", body, "got", w.Body.String())
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
		auth.CurrentUser{UserName: "username", ID: 1})
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))

	// Add our context to the request
	r = r.WithContext(ctx)

	typeRef := tester{ID: 1}
	updateFunc := UpdateHandler(GetTypeSingleton())

	//verifies we get the right changelog insertion
	expectedMessage := Updated + " " + typeRef.GetType() + ": " + typeRef.GetAuditName() + " keys: { id:1 }"
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WithArgs(ApiChange, expectedMessage, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	updateFunc(w, r)

	//verifies the body is in the expected format
	body := `{"alerts":[{"text":"tester was updated.","level":"success"}],"response":{"ID":1}}`
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
		auth.CurrentUser{UserName: "username", ID: 1})
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{"id": "1"})
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, &cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(0))
	// Add our context to the request
	r = r.WithContext(ctx)

	typeRef := tester{ID: 1}
	deleteFunc := DeleteHandler(GetTypeSingleton())

	//verifies we get the right changelog insertion
	expectedMessage := Deleted + " " + typeRef.GetType() + ": " + typeRef.GetAuditName() + " keys: { id:1 }"
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WithArgs(ApiChange, expectedMessage, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	deleteFunc(w, r)

	//verifies the body is in the expected format
	body := `{"alerts":[{"text":"tester was deleted.","level":"success"}]}`
	if w.Body.String() != body {
		t.Error("Expected body", body, "got", w.Body.String())
	}
}
