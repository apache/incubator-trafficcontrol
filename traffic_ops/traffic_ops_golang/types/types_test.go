package types

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
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/test"
	"github.com/jmoiron/sqlx"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func getTestTypes() []tc.TypeNullable {
	types := []tc.TypeNullable{}
	ID := 1
	name := "name1"
	description := "desc"
	useInTable := "use_in_table1"
	lastUpdated := tc.TimeNoMod{Time: time.Now()}
	testCase := tc.TypeNullable{
		ID:          &ID,
		LastUpdated: &lastUpdated,
		Name:        &name,
		Description: &description,
		UseInTable:  &useInTable,
	}
	types = append(types, testCase)

	testCase2 := testCase
	name = "name2"
	testCase2.Name = &name
	types = append(types, testCase2)

	return types
}

func TestGetType(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	testCase := getTestTypes()
	cols := test.ColsFromStructByTag("db", tc.TypeNullable{})
	rows := sqlmock.NewRows(cols)

	for _, ts := range testCase {
		rows = rows.AddRow(
			ts.ID,
			ts.LastUpdated,
			ts.Name,
			ts.Description,
			ts.UseInTable,
		)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	mock.ExpectCommit()

	reqInfo := api.Info{Tx: db.MustBegin(), Params: map[string]string{"dsId": "1"}}

	obj := TOType{
		api.InfoerImpl{ReqInfo: &reqInfo},
		tc.TypeNullable{},
	}
	types, userErr, sysErr, _, _ := obj.Read(nil, false)
	if userErr != nil || sysErr != nil {
		t.Errorf("Read expected: no errors, actual: %v %v", userErr, sysErr)
	}

	if len(types) != 2 {
		t.Errorf("type.Read expected: len(types) == 2, actual: %v", len(types))
	}

}

func TestInterfaces(t *testing.T) {
	var i interface{}
	i = &TOType{}

	if _, ok := i.(api.Creator); !ok {
		t.Errorf("Type must be Creator")
	}
	if _, ok := i.(api.Reader); !ok {
		t.Errorf("Type must be Reader")
	}
	if _, ok := i.(api.Updater); !ok {
		t.Errorf("Type must be Updater")
	}
	if _, ok := i.(api.Deleter); !ok {
		t.Errorf("Type must be Deleter")
	}
	if _, ok := i.(api.Identifier); !ok {
		t.Errorf("Type must be Identifier")
	}
}

func createDummyType(field string) *TOType {
	version := api.Version{
		Major: 2,
		Minor: 0,
	}
	apiInfo := api.Info{
		Version: version,
	}
	return &TOType{
		TypeNullable: tc.TypeNullable{
			Name:        &field,
			Description: &field,
			UseInTable:  &field,
		},
		InfoerImpl: api.InfoerImpl{
			ReqInfo: &apiInfo,
		},
	}
}

func TestUpdateInvalidType(t *testing.T) {
	invalidUpdateType := createDummyType("test")

	err, _, statusCode := invalidUpdateType.Update(nil)
	if err == nil {
		t.Fatalf("expected update type tp have an error")
	}
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected update to return a 400 error")
	}
}

func TestDeleteInvalidType(t *testing.T) {
	invalidDeleteType := createDummyType("other")

	err, _, statusCode := invalidDeleteType.Delete()
	if err == nil {
		t.Fatalf("expected delete type to have an error")
	}
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected delete type to return a %v error", http.StatusBadRequest)
	}
}

func TestCreateInvalidType(t *testing.T) {
	invalidCreateType := createDummyType("test")

	err, _, statusCode := invalidCreateType.Create()
	if err == nil {
		t.Fatalf("expected create type to have an error")
	}
	if statusCode != http.StatusBadRequest {
		t.Fatalf("expected create type to return a %v error", http.StatusBadRequest)
	}
}

func TestValidate(t *testing.T) {
	p := TOType{}
	errs := util.JoinErrsStr(test.SortErrors(test.SplitErrors(p.Validate())))
	expected := util.JoinErrsStr(test.SortErrors([]error{
		errors.New("'name' cannot be blank"),
		errors.New("'description' cannot be blank"),
		errors.New("'use_in_table' cannot be blank"),
	}))

	if !reflect.DeepEqual(expected, errs) {
		t.Errorf("expected %++v,  got %++v", expected, errs)
	}
}
